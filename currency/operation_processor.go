package currency

import (
	"sync"

	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util/valuehash"
)

type OperationProcessor struct {
	sync.RWMutex
	pool             *isaac.Statepool
	amountPool       map[string]*AmountState
	processedSenders map[string]struct{}
}

func (opr *OperationProcessor) New(pool *isaac.Statepool) isaac.OperationProcessor {
	return &OperationProcessor{
		pool:             pool,
		amountPool:       map[string]*AmountState{},
		processedSenders: map[string]struct{}{},
	}
}

func (opr *OperationProcessor) getState(key string) (state.StateUpdater, bool, error) {
	opr.Lock()
	defer opr.Unlock()

	if ast, found := opr.amountPool[key]; found {
		return ast, true, nil
	} else if st, found, err := opr.pool.Get(key); err != nil {
		return nil, false, err
	} else {
		ast := NewAmountState(st)
		opr.amountPool[key] = ast

		return ast, found, nil
	}
}

func (opr *OperationProcessor) setState(oph valuehash.Hash, s ...state.StateUpdater) error {
	ns := make([]state.StateUpdater, len(s))
	for i := range s {
		if u, ok := s[i].(*AmountState); ok {
			ns[i] = u.StateUpdater
		} else {
			ns[i] = s[i]
		}
	}

	return opr.pool.Set(oph, ns...)
}

func (opr *OperationProcessor) PreProcess(op state.Processor) (state.Processor, error) {
	var sp state.Processor
	var sender string
	var get func(string) (state.StateUpdater, bool, error)
	var set func(valuehash.Hash, ...state.StateUpdater) error

	switch t := op.(type) {
	case Transfer:
		get = opr.getState
		set = opr.setState
		sp = &TransferProcessor{Transfer: t}
		sender = t.Fact().(TransferFact).Sender().String()
	case CreateAccount:
		get = opr.getState
		set = opr.setState
		sp = &CreateAccountProcessor{CreateAccount: t}
		sender = t.Fact().(CreateAccountFact).Sender().String()
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

	if pr, err := sp.(state.PreProcessor).PreProcess(get, set); err != nil {
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
	case *TransferProcessor, *CreateAccountProcessor:
		return opr.process(op)
	case Transfer, CreateAccount:
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
	var get func(string) (state.StateUpdater, bool, error)
	var set func(valuehash.Hash, ...state.StateUpdater) error

	switch t := op.(type) {
	case *TransferProcessor:
		get = opr.getState
		set = opr.setState
		sp = t
	case *CreateAccountProcessor:
		get = opr.getState
		set = opr.setState
		sp = t
	default:
		return op.Process(opr.pool.Get, opr.pool.Set)
	}

	return sp.Process(get, set)
}
