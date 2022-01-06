package currency

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/valuehash"
)

var keyUpdaterProcessorPool = sync.Pool{
	New: func() interface{} {
		return new(KeyUpdaterProcessor)
	},
}

func (KeyUpdater) Process(
	func(key string) (state.State, bool, error),
	func(valuehash.Hash, ...state.State) error,
) error {
	return nil
}

type KeyUpdaterProcessor struct {
	cp *CurrencyPool
	KeyUpdater
	sa  state.State
	sb  AmountState
	fee Big
}

func NewKeyUpdaterProcessor(cp *CurrencyPool) GetNewProcessor {
	return func(op state.Processor) (state.Processor, error) {
		i, ok := op.(KeyUpdater)
		if !ok {
			return nil, errors.Errorf("not KeyUpdater, %T", op)
		}

		opp := keyUpdaterProcessorPool.Get().(*KeyUpdaterProcessor)

		opp.cp = cp
		opp.KeyUpdater = i
		opp.sa = nil
		opp.sb = AmountState{}
		opp.fee = ZeroBig

		return opp, nil
	}
}

func (opp *KeyUpdaterProcessor) PreProcess(
	getState func(string) (state.State, bool, error),
	_ func(valuehash.Hash, ...state.State) error,
) (state.Processor, error) {
	fact := opp.Fact().(KeyUpdaterFact)

	st, err := existsState(StateKeyAccount(fact.target), "target keys", getState)
	if err != nil {
		return nil, err
	}
	opp.sa = st

	if ks, e := StateKeysValue(opp.sa); err != nil {
		return nil, operation.NewBaseReasonErrorFromError(e)
	} else if ks.Equal(fact.Keys()) {
		return nil, operation.NewBaseReasonError("same Keys with the existing")
	}

	st, err = existsState(StateKeyBalance(fact.target, fact.currency), "balance of target", getState)
	if err != nil {
		return nil, err
	}
	opp.sb = NewAmountState(st, fact.currency)

	if err = checkFactSignsByState(fact.target, opp.Signs(), getState); err != nil {
		return nil, errors.Wrap(err, "invalid signing")
	}

	feeer, found := opp.cp.Feeer(fact.currency)
	if !found {
		return nil, operation.NewBaseReasonError("currency, %q not found of KeyUpdater", fact.currency)
	}

	fee, err := feeer.Fee(ZeroBig)
	if err != nil {
		return nil, operation.NewBaseReasonErrorFromError(err)
	}
	switch b, err := StateBalanceValue(opp.sb); {
	case err != nil:
		return nil, operation.NewBaseReasonErrorFromError(err)
	case b.Big().Compare(fee) < 0:
		return nil, operation.NewBaseReasonError("insufficient balance with fee")
	default:
		opp.fee = fee
	}

	return opp, nil
}

func (opp *KeyUpdaterProcessor) Process(
	_ func(key string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) error {
	fact := opp.Fact().(KeyUpdaterFact)

	opp.sb = opp.sb.Sub(opp.fee).AddFee(opp.fee)
	st, err := SetStateKeysValue(opp.sa, fact.keys)
	if err != nil {
		return operation.NewBaseReasonErrorFromError(err)
	}
	return setState(fact.Hash(), st, opp.sb)
}

func (opp *KeyUpdaterProcessor) Close() error {
	opp.cp = nil
	opp.KeyUpdater = KeyUpdater{}
	opp.sa = nil
	opp.sb = AmountState{}
	opp.fee = ZeroBig

	keyUpdaterProcessorPool.Put(opp)

	return nil
}
