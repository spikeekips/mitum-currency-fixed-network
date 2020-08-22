package currency

import (
	"sync"

	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/isaac"
)

type OperationProcessor struct {
	sync.RWMutex
	pool             *isaac.Statepool
	amountPool       map[string]AmountState
	processedSenders map[string]struct{}
}

func (opr *OperationProcessor) New(pool *isaac.Statepool) isaac.OperationProcessor {
	return &OperationProcessor{
		pool:             pool,
		amountPool:       map[string]AmountState{},
		processedSenders: map[string]struct{}{},
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
	default:
		return op, nil
	}

	if func() bool {
		opr.RLock()
		defer opr.RUnlock()

		_, found := opr.processedSenders[sender]

		return found
	}() {
		return nil, state.IgnoreOperationProcessingError.Errorf("violates only one sender in proposal")
	}

	if pr, err := sp.(state.PreProcessor).PreProcess(get, opr.pool.Set); err != nil {
		return nil, err
	} else {
		opr.Lock()
		opr.processedSenders[sender] = struct{}{}
		opr.Unlock()

		return pr, nil
	}
}

func (opr *OperationProcessor) Process(op state.Processor) error {
	switch op.(type) {
	case *TransfersProcessor, *CreateAccountsProcessor:
		return opr.process(op)
	case Transfers, CreateAccounts:
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
	default:
		return op.Process(opr.pool.Get, opr.pool.Set)
	}

	return sp.Process(get, opr.pool.Set)
}
