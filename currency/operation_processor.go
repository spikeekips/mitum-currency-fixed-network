package currency

import (
	"sync"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util/logging"
)

type OperationProcessor struct {
	sync.RWMutex
	*logging.Logging
	pool                *isaac.Statepool
	amountPool          map[string]AmountState
	processedSenders    map[string]struct{}
	processedNewAddress map[string]struct{}
}

func (opr *OperationProcessor) New(pool *isaac.Statepool) isaac.OperationProcessor {
	return &OperationProcessor{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "mitum-currency-operations-processor")
		}),
		pool:                pool,
		amountPool:          map[string]AmountState{},
		processedSenders:    map[string]struct{}{},
		processedNewAddress: map[string]struct{}{},
	}
}

func (opr *OperationProcessor) getState(key string) (state.State, bool, error) {
	opr.Lock()
	defer opr.Unlock()

	if ast, found := opr.amountPool[key]; found {
		return ast, ast.exists, nil
	} else if st, exists, err := opr.pool.Get(key); err != nil {
		return nil, false, err
	} else {
		ast := NewAmountState(st, exists)
		opr.amountPool[key] = ast

		return ast, ast.exists, nil
	}
}

func (opr *OperationProcessor) PreProcess(op state.Processor) (state.Processor, error) {
	var sp state.Processor
	var sender string
	var get func(string) (state.State, bool, error)

	switch t := op.(type) {
	case Transfers:
		get = opr.getState
		sp = &TransfersProcessor{Transfers: t}
		sender = t.Fact().(TransfersFact).Sender().String()
	case CreateAccounts:
		get = opr.getState
		sp = &CreateAccountsProcessor{CreateAccounts: t}
		sender = t.Fact().(CreateAccountsFact).Sender().String()
	case KeyUpdater:
		get = opr.pool.Get
		sp = &KeyUpdaterProcessor{KeyUpdater: t}
		sender = t.Fact().(KeyUpdaterFact).Target().String()
	default:
		return op, nil
	}

	var pop state.Processor
	if pr, err := sp.(state.PreProcessor).PreProcess(get, opr.pool.Set); err != nil {
		return nil, err
	} else {
		pop = pr
	}

	if func() bool {
		opr.RLock()
		defer opr.RUnlock()

		_, found := opr.processedSenders[sender]

		return found
	}() {
		return nil, state.IgnoreOperationProcessingError.Errorf("violates only one sender in proposal")
	}

	if t, ok := op.(CreateAccounts); ok {
		fact := t.Fact().(CreateAccountsFact)
		if as, err := fact.Addresses(); err != nil {
			return nil, state.IgnoreOperationProcessingError.Errorf("failed to get Addresses")
		} else if opr.checkNewAddresses(as) {
			return nil, state.IgnoreOperationProcessingError.Errorf("new address already processed")
		}
	}

	opr.Lock()
	opr.processedSenders[sender] = struct{}{}
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
	var get func(string) (state.State, bool, error)

	switch t := op.(type) {
	case *TransfersProcessor:
		get = opr.getState
		sp = t
	case *CreateAccountsProcessor:
		get = opr.getState
		sp = t
	case *KeyUpdaterProcessor:
		sp = t
	default:
		return op.Process(opr.pool.Get, opr.pool.Set)
	}

	return sp.Process(get, opr.pool.Set)
}

func (opr *OperationProcessor) checkNewAddresses(as []base.Address) bool {
	opr.Lock()
	defer opr.Unlock()

	for i := range as {
		if _, found := opr.processedNewAddress[as[i].String()]; found {
			return true
		}
	}

	for i := range as {
		opr.processedNewAddress[as[i].String()] = struct{}{}
	}

	return false
}
