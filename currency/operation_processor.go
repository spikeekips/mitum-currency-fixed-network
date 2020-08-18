package currency

import (
	"sync"

	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util/valuehash"
)

type PreProcessor interface {
	PreProcess(
		getState func(key string) (state.StateUpdater, bool, error),
		setState func(valuehash.Hash, ...state.StateUpdater) error,
	) error
}

type OperationProcessor struct {
	sync.RWMutex
	pool            *isaac.Statepool
	amountPool      map[string]*AmountState
	processedSender map[string]struct{}
}

func (opr *OperationProcessor) New(pool *isaac.Statepool) isaac.OperationProcessor {
	return &OperationProcessor{
		pool:            pool,
		amountPool:      map[string]*AmountState{},
		processedSender: map[string]struct{}{},
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

func (opr *OperationProcessor) Process(op state.StateProcessor) error {
	var sp state.StateProcessor
	var sender string
	var get func(string) (state.StateUpdater, bool, error)

	switch t := op.(type) {
	case Transfer:
		get = opr.getState
		sp = &TransferProcessor{Transfer: t}
		sender = t.Fact().(TransferFact).Sender().String()
	case CreateAccount:
		get = opr.getState
		sp = &CreateAccountProcessor{CreateAccount: t}
		sender = t.Fact().(CreateAccountFact).Sender().String()
	default:
		return sp.Process(opr.pool.Get, opr.pool.Set)
	}

	if func() bool {
		opr.RLock()
		defer opr.RUnlock()

		_, found := opr.processedSender[sender]

		return found
	}() {
		return state.IgnoreOperationProcessingError.Errorf("violates only one sender in proposal")
	}

	if pr, ok := sp.(PreProcessor); ok {
		if err := pr.PreProcess(get, opr.pool.Set); err != nil {
			return err
		}
	}

	if err := sp.Process(get, opr.pool.Set); err != nil {
		return err
	} else {
		opr.Lock()
		defer opr.Unlock()

		opr.processedSender[sender] = struct{}{}

		return nil
	}
}
