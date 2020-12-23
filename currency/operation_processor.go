package currency

import (
	"sync"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

type OperationProcessor struct {
	sync.RWMutex
	*logging.Logging
	feeAmount           FeeAmount
	getFeeReceiver      func() (base.Address, error)
	feeReceiver         base.Address
	pool                *storage.Statepool
	fee                 Amount
	amountPool          map[string]AmountState
	processedSenders    map[string]struct{}
	processedNewAddress map[string]struct{}
}

func NewOperationProcessor(feeAmount FeeAmount, getFeeReceiver func() (base.Address, error)) *OperationProcessor {
	if getFeeReceiver == nil {
		feeAmount = NewNilFeeAmount()
	}

	return &OperationProcessor{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "mitum-currency-operations-processor")
		}),
		feeAmount:      feeAmount,
		getFeeReceiver: getFeeReceiver,
	}
}

func (opr *OperationProcessor) New(pool *storage.Statepool) prprocessor.OperationProcessor {
	var feeReceiver base.Address
	if opr.getFeeReceiver != nil {
		if a, err := opr.getFeeReceiver(); err == nil {
			feeReceiver = a
		}
	}

	return &OperationProcessor{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "mitum-currency-operations-processor")
		}),
		feeAmount:           opr.feeAmount,
		feeReceiver:         feeReceiver,
		pool:                pool,
		fee:                 ZeroAmount,
		amountPool:          map[string]AmountState{},
		processedSenders:    map[string]struct{}{},
		processedNewAddress: map[string]struct{}{},
	}
}

func (opr *OperationProcessor) setState(op valuehash.Hash, sts ...state.State) error {
	opr.Lock()
	defer opr.Unlock()

	sum := NewAmount(0)
	for i := range sts {
		if t, ok := sts[i].(AmountState); ok {
			if t.Fee().Compare(ZeroAmount) <= 0 {
				continue
			} else {
				sum = sum.Add(t.Fee())
			}
		}
	}

	opr.fee = opr.fee.Add(sum)

	return opr.pool.Set(op, sts...)
}

func (opr *OperationProcessor) PreProcess(op state.Processor) (state.Processor, error) {
	var sp state.Processor
	var sender string
	var addresses []base.Address

	switch t := op.(type) {
	case Transfers:
		fact := t.Fact().(TransfersFact)
		sender = fact.Sender().String()
		sp = &TransfersProcessor{Transfers: t, fa: opr.feeAmount}
	case CreateAccounts:
		fact := t.Fact().(CreateAccountsFact)
		if as, err := fact.Targets(); err != nil {
			return nil, util.IgnoreError.Errorf("failed to get Addresses")
		} else if err := opr.checkNewAddressDuplication(as); err != nil {
			return nil, err
		} else {
			addresses = as
		}

		sender = fact.Sender().String()
		sp = &CreateAccountsProcessor{CreateAccounts: t, fa: opr.feeAmount}
	case KeyUpdater:
		sender = t.Fact().(KeyUpdaterFact).Target().String()
		sp = &KeyUpdaterProcessor{KeyUpdater: t, fa: opr.feeAmount}
	default:
		return op, nil
	}

	if err := opr.checkSenderDuplication(sender); err != nil {
		return nil, err
	}

	var pop state.Processor
	if pr, err := sp.(state.PreProcessor).PreProcess(opr.pool.Get, opr.setState); err != nil {
		return nil, err
	} else {
		pop = pr
	}

	opr.Lock()
	opr.processedSenders[sender] = struct{}{}
	for i := range addresses {
		opr.processedNewAddress[addresses[i].String()] = struct{}{}
	}
	opr.Unlock()

	return pop, nil
}

func (opr *OperationProcessor) Process(op state.Processor) error {
	switch op.(type) {
	case *TransfersProcessor, *CreateAccountsProcessor, *KeyUpdaterProcessor:
		return opr.process(op)
	case Transfers, CreateAccounts, KeyUpdater:
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

func (opr *OperationProcessor) checkSenderDuplication(sender string) error {
	opr.RLock()
	defer opr.RUnlock()

	if _, found := opr.processedSenders[sender]; found {
		return util.IgnoreError.Errorf("violates only one sender in proposal")
	}

	return nil
}

func (opr *OperationProcessor) checkNewAddressDuplication(as []base.Address) error {
	opr.Lock()
	defer opr.Unlock()

	for i := range as {
		if _, found := opr.processedNewAddress[as[i].String()]; found {
			return util.IgnoreError.Errorf("new address already processed")
		}
	}

	return nil
}

func (opr *OperationProcessor) Close() error {
	opr.RLock()
	defer opr.RUnlock()

	if opr.feeReceiver != nil {
		if opr.fee.Compare(ZeroAmount) < 1 {
			return nil
		}

		fact := NewFeeOperationFact(opr.feeAmount, opr.pool.Height(), opr.feeReceiver, opr.fee)
		op := NewFeeOperation(fact)
		if err := op.Process(opr.pool.Get, opr.pool.Set); err != nil {
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
