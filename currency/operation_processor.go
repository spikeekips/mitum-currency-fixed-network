package currency

import (
	"fmt"
	"io"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

var operationProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(OperationProcessor)
	},
}

type GetNewProcessor func(state.Processor) (state.Processor, error)

type DuplicationType string

const (
	DuplicationTypeSender   DuplicationType = "sender"
	DuplicationTypeCurrency DuplicationType = "currency"
)

type OperationProcessor struct {
	id string
	sync.RWMutex
	*logging.Logging
	processorHintSet     *hint.Hintmap
	cp                   *CurrencyPool
	pool                 *storage.Statepool
	fee                  map[CurrencyID]Big
	amountPool           map[string]AmountState
	duplicated           map[string]DuplicationType
	duplicatedNewAddress map[string]struct{}
	processorClosers     *sync.Map
}

func NewOperationProcessor(cp *CurrencyPool) *OperationProcessor {
	return &OperationProcessor{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "mitum-currency-operations-processor")
		}),
		processorHintSet: hint.NewHintmap(),
		cp:               cp,
	}
}

func (opr *OperationProcessor) New(pool *storage.Statepool) prprocessor.OperationProcessor {
	nopr := operationProcessorPool.Get().(*OperationProcessor)
	nopr.id = util.UUID().String()

	if nopr.Logging == nil {
		nopr.Logging = logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "mitum-currency-operations-processor")
		})
		_ = nopr.SetLogging(opr.Logging)
	}

	if nopr.processorHintSet == nil {
		nopr.processorHintSet = opr.processorHintSet
	}

	if nopr.cp == nil {
		nopr.cp = opr.cp
	}

	nopr.pool = pool
	nopr.fee = map[CurrencyID]Big{}
	nopr.amountPool = map[string]AmountState{}
	nopr.duplicated = map[string]DuplicationType{}
	nopr.duplicatedNewAddress = map[string]struct{}{}
	nopr.processorClosers = &sync.Map{}

	nopr.Log().Debug().Str("processor_id", nopr.id).Msg("new operation processors created")

	return nopr
}

func (opr *OperationProcessor) SetProcessor(
	hinter hint.Hinter,
	newProcessor GetNewProcessor,
) (prprocessor.OperationProcessor, error) {
	if err := opr.processorHintSet.Add(hinter, newProcessor); err != nil {
		if !errors.Is(err, util.FoundError) {
			return nil, err
		}
	}

	return opr, nil
}

func (opr *OperationProcessor) setState(op valuehash.Hash, sts ...state.State) error {
	opr.Lock()
	defer opr.Unlock()

	for i := range sts {
		if t, ok := sts[i].(AmountState); ok {
			if t.Fee().OverZero() {
				f := ZeroBig
				if i, found := opr.fee[t.Currency()]; found {
					f = i
				}

				opr.fee[t.Currency()] = f.Add(t.Fee())
			}
		}
	}

	return opr.pool.Set(op, sts...)
}

func (opr *OperationProcessor) PreProcess(op state.Processor) (state.Processor, error) {
	var sp state.Processor
	switch i, known, err := opr.getNewProcessor(op); {
	case err != nil:
		return nil, operation.NewBaseReasonErrorFromError(err)
	case !known:
		return op, nil
	default:
		sp = i
	}

	pop, err := sp.(state.PreProcessor).PreProcess(opr.pool.Get, opr.setState)
	if err != nil {
		return nil, err
	}

	if err := opr.checkDuplication(op); err != nil {
		return nil, operation.NewBaseReasonError("duplication found: %w", err)
	}

	return pop, nil
}

func (opr *OperationProcessor) Process(op state.Processor) error {
	switch op.(type) {
	case *TransfersProcessor,
		*CreateAccountsProcessor,
		*KeyUpdaterProcessor,
		*CurrencyRegisterProcessor,
		*CurrencyPolicyUpdaterProcessor,
		*SuffrageInflationProcessor:
		return opr.process(op)
	case Transfers, CreateAccounts, KeyUpdater, CurrencyRegister, CurrencyPolicyUpdater, SuffrageInflation:
		pr, err := opr.PreProcess(op)
		if err != nil {
			return err
		}
		return opr.process(pr)
	default:
		return op.Process(opr.pool.Get, opr.pool.Set)
	}
}

func (opr *OperationProcessor) process(op state.Processor) error {
	var sp state.Processor

	switch t := op.(type) {
	case *TransfersProcessor:
		sp = t
	case *CreateAccountsProcessor:
		sp = t
	case *KeyUpdaterProcessor:
		sp = t
	default:
		return op.Process(opr.pool.Get, opr.pool.Set)
	}

	return sp.Process(opr.pool.Get, opr.setState)
}

func (opr *OperationProcessor) checkDuplication(op state.Processor) error {
	opr.Lock()
	defer opr.Unlock()

	var did string
	var didtype DuplicationType
	var newAddresses []base.Address

	switch t := op.(type) {
	case Transfers:
		did = t.Fact().(TransfersFact).Sender().String()
		didtype = DuplicationTypeSender
	case CreateAccounts:
		fact := t.Fact().(CreateAccountsFact)
		as, err := fact.Targets()
		if err != nil {
			return errors.Errorf("failed to get Addresses")
		}
		newAddresses = as

		did = fact.Sender().String()
		didtype = DuplicationTypeSender
	case KeyUpdater:
		did = t.Fact().(KeyUpdaterFact).Target().String()
		didtype = DuplicationTypeSender
	case CurrencyRegister:
		did = t.Fact().(CurrencyRegisterFact).Currency().Currency().String()
		didtype = DuplicationTypeCurrency
	case CurrencyPolicyUpdater:
		did = t.Fact().(CurrencyPolicyUpdaterFact).Currency().String()
		didtype = DuplicationTypeCurrency
	default:
		return nil
	}

	if len(did) > 0 {
		if _, found := opr.duplicated[did]; found {
			switch didtype {
			case DuplicationTypeSender:
				return errors.Errorf("violates only one sender in proposal")
			case DuplicationTypeCurrency:
				return errors.Errorf("duplicated currency id, %q found in proposal", did)
			default:
				return errors.Errorf("violates duplication in proposal")
			}
		}

		opr.duplicated[did] = didtype
	}

	if len(newAddresses) > 0 {
		if err := opr.checkNewAddressDuplication(newAddresses); err != nil {
			return err
		}
	}

	return nil
}

func (opr *OperationProcessor) checkNewAddressDuplication(as []base.Address) error {
	for i := range as {
		if _, found := opr.duplicatedNewAddress[as[i].String()]; found {
			return errors.Errorf("new address already processed")
		}
	}

	for i := range as {
		opr.duplicatedNewAddress[as[i].String()] = struct{}{}
	}

	return nil
}

func (opr *OperationProcessor) Close() error {
	opr.Lock()
	defer opr.Unlock()

	defer opr.close()

	if opr.cp != nil && len(opr.fee) > 0 {
		op := NewFeeOperation(NewFeeOperationFact(opr.pool.Height(), opr.fee))

		pr := NewFeeOperationProcessor(opr.cp, op)
		if err := pr.Process(opr.pool.Get, opr.pool.Set); err != nil {
			return err
		}
		opr.pool.AddOperations(op)
	}

	return nil
}

func (opr *OperationProcessor) Cancel() error {
	opr.Lock()
	defer opr.Unlock()

	defer opr.close()

	return nil
}

func (opr *OperationProcessor) getNewProcessor(op state.Processor) (state.Processor, bool, error) {
	switch i, err := opr.getNewProcessorFromHintset(op); {
	case err != nil:
		return nil, false, err
	case i != nil:
		return i, true, nil
	}

	switch t := op.(type) {
	case Transfers,
		CreateAccounts,
		KeyUpdater,
		CurrencyRegister,
		CurrencyPolicyUpdater,
		SuffrageInflation:
		return nil, false, errors.Errorf("%T needs SetProcessor", t)
	default:
		return op, false, nil
	}
}

func (opr *OperationProcessor) getNewProcessorFromHintset(op state.Processor) (state.Processor, error) {
	var f GetNewProcessor
	if hinter, ok := op.(hint.Hinter); !ok {
		return nil, nil
	} else if i, err := opr.processorHintSet.Compatible(hinter); err != nil {
		if errors.Is(err, util.NotFoundError) {
			return nil, nil
		}

		return nil, err
	} else if j, ok := i.(GetNewProcessor); !ok {
		return nil, errors.Errorf("invalid GetNewProcessor func, %T", i)
	} else {
		f = j
	}

	opp, err := f(op)
	if err != nil {
		return nil, err
	}

	h := op.(valuehash.Hasher).Hash().String()
	_, iscloser := opp.(io.Closer)
	if iscloser {
		opr.processorClosers.Store(h, opp)
		iscloser = true
	}

	opr.Log().Debug().
		Str("operation", h).
		Str("processor", fmt.Sprintf("%T", opp)).
		Bool("is_closer", iscloser).
		Msg("operation processor created")

	return opp, nil
}

func (opr *OperationProcessor) close() {
	opr.processorClosers.Range(func(_, v interface{}) bool {
		err := v.(io.Closer).Close()
		if err != nil {
			opr.Log().Error().Err(err).Str("op", fmt.Sprintf("%T", v)).Msg("failed to close operation processor")
		} else {
			opr.Log().Debug().Str("processor", fmt.Sprintf("%T", v)).Msg("operation processor closed")
		}

		return true
	})

	opr.pool = nil
	opr.fee = nil
	opr.amountPool = nil
	opr.duplicated = nil
	opr.duplicatedNewAddress = nil
	opr.processorClosers = nil

	operationProcessorPool.Put(opr)

	opr.Log().Debug().Str("processor_id", opr.id).Msg("operation processors closed")
}
