package currency

import (
	"sync"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

type GetNewProcessor func(state.Processor) (state.Processor, error)

type DuplicationType string

const (
	DuplicationTypeSender   DuplicationType = "sender"
	DuplicationTypeCurrency DuplicationType = "currency"
)

type OperationProcessor struct {
	sync.RWMutex
	*logging.Logging
	processorHintSet     *hint.Hintmap
	cp                   *CurrencyPool
	pool                 *storage.Statepool
	fee                  map[CurrencyID]Big
	amountPool           map[string]AmountState
	duplicated           map[string]DuplicationType
	duplicatedNewAddress map[string]struct{}
}

func NewOperationProcessor(cp *CurrencyPool) *OperationProcessor {
	return &OperationProcessor{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "mitum-currency-operations-processor")
		}),
		processorHintSet: hint.NewHintmap(),
		cp:               cp,
	}
}

func (opr *OperationProcessor) New(pool *storage.Statepool) prprocessor.OperationProcessor {
	return &OperationProcessor{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "mitum-currency-operations-processor")
		}),
		processorHintSet:     opr.processorHintSet,
		cp:                   opr.cp,
		pool:                 pool,
		fee:                  map[CurrencyID]Big{},
		amountPool:           map[string]AmountState{},
		duplicated:           map[string]DuplicationType{},
		duplicatedNewAddress: map[string]struct{}{},
	}
}

func (opr *OperationProcessor) SetProcessor(
	hinter hint.Hinter,
	newProcessor GetNewProcessor,
) (prprocessor.OperationProcessor, error) {
	if err := opr.processorHintSet.Add(hinter, newProcessor); err != nil {
		return nil, err
	} else {
		return opr, nil
	}
}

func (opr *OperationProcessor) setState(op valuehash.Hash, sts ...state.State) error {
	opr.Lock()
	defer opr.Unlock()

	for i := range sts {
		if t, ok := sts[i].(AmountState); ok {
			if t.Fee().OverZero() {
				var f Big = ZeroBig
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
		return nil, util.IgnoreError.Wrap(err)
	case !known:
		return op, nil
	default:
		sp = i
	}

	var pop state.Processor
	if pr, err := sp.(state.PreProcessor).PreProcess(opr.pool.Get, opr.setState); err != nil {
		return nil, err
	} else {
		pop = pr
	}

	if err := opr.checkDuplication(op); err != nil {
		return nil, util.IgnoreError.Errorf("duplication found: %w", err)
	}

	return pop, nil
}

func (opr *OperationProcessor) Process(op state.Processor) error {
	switch op.(type) {
	case *TransfersProcessor,
		*CreateAccountsProcessor,
		*KeyUpdaterProcessor,
		*CurrencyRegisterProcessor,
		*CurrencyPolicyUpdaterProcessor:
		return opr.process(op)
	case Transfers, CreateAccounts, KeyUpdater, CurrencyRegister, CurrencyPolicyUpdater:
		if pr, err := opr.PreProcess(op); err != nil {
			return err
		} else {
			return opr.process(pr)
		}
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
		if as, err := fact.Targets(); err != nil {
			return xerrors.Errorf("failed to get Addresses")
		} else {
			newAddresses = as
		}

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
				return xerrors.Errorf("violates only one sender in proposal")
			case DuplicationTypeCurrency:
				return xerrors.Errorf("duplicated currency id, %q found in proposal", did)
			default:
				return xerrors.Errorf("violates duplication in proposal")
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
			return xerrors.Errorf("new address already processed")
		}
	}

	for i := range as {
		opr.duplicatedNewAddress[as[i].String()] = struct{}{}
	}

	return nil
}

func (opr *OperationProcessor) Close() error {
	opr.RLock()
	defer opr.RUnlock()

	if opr.cp != nil && len(opr.fee) > 0 {
		op := NewFeeOperation(NewFeeOperationFact(opr.pool.Height(), opr.fee))

		pr := NewFeeOperationProcessor(opr.cp, op)
		if err := pr.Process(opr.pool.Get, opr.pool.Set); err != nil {
			return err
		} else {
			opr.pool.AddOperations(op)
		}
	}

	return nil
}

func (opr *OperationProcessor) Cancel() error {
	opr.RLock()
	defer opr.RUnlock()

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
		CurrencyPolicyUpdater:
		return nil, false, xerrors.Errorf("%T needs SetProcessor", t)
	default:
		return op, false, nil
	}
}

func (opr *OperationProcessor) getNewProcessorFromHintset(op state.Processor) (state.Processor, error) {
	var f GetNewProcessor
	if hinter, ok := op.(hint.Hinter); !ok {
		return nil, nil
	} else if i, found := opr.processorHintSet.Get(hinter); !found {
		return nil, nil
	} else if j, ok := i.(GetNewProcessor); !ok {
		return nil, xerrors.Errorf("invalid GetNewProcessor func, %%", i)
	} else {
		f = j
	}

	return f(op)
}
