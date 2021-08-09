package currency

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/valuehash"
)

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
		return &KeyUpdaterProcessor{
			cp:         cp,
			KeyUpdater: i,
		}, nil
	}
}

func (op *KeyUpdaterProcessor) PreProcess(
	getState func(string) (state.State, bool, error),
	_ func(valuehash.Hash, ...state.State) error,
) (state.Processor, error) {
	fact := op.Fact().(KeyUpdaterFact)

	st, err := existsState(StateKeyAccount(fact.target), "target keys", getState)
	if err != nil {
		return nil, err
	}
	op.sa = st

	if ks, e := StateKeysValue(op.sa); err != nil {
		return nil, operation.NewBaseReasonErrorFromError(e)
	} else if ks.Equal(fact.Keys()) {
		return nil, operation.NewBaseReasonError("same Keys with the existing")
	}

	st, err = existsState(StateKeyBalance(fact.target, fact.currency), "balance of target", getState)
	if err != nil {
		return nil, err
	}
	op.sb = NewAmountState(st, fact.currency)

	if err = checkFactSignsByState(fact.target, op.Signs(), getState); err != nil {
		return nil, operation.NewBaseReasonError("invalid signing: %w", err)
	}

	feeer, found := op.cp.Feeer(fact.currency)
	if !found {
		return nil, operation.NewBaseReasonError("currency, %q not found of KeyUpdater", fact.currency)
	}

	fee, err := feeer.Fee(ZeroBig)
	if err != nil {
		return nil, operation.NewBaseReasonErrorFromError(err)
	}
	switch b, err := StateBalanceValue(op.sb); {
	case err != nil:
		return nil, operation.NewBaseReasonErrorFromError(err)
	case b.Big().Compare(fee) < 0:
		return nil, operation.NewBaseReasonError("insufficient balance with fee")
	default:
		op.fee = fee
	}

	return op, nil
}

func (op *KeyUpdaterProcessor) Process(
	_ func(key string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) error {
	fact := op.Fact().(KeyUpdaterFact)

	op.sb = op.sb.Sub(op.fee).AddFee(op.fee)
	st, err := SetStateKeysValue(op.sa, fact.keys)
	if err != nil {
		return err
	}
	return setState(fact.Hash(), st, op.sb)
}
