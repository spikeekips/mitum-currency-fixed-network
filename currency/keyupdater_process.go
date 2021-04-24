package currency

import (
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

func (op KeyUpdater) Process(
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
		if i, ok := op.(KeyUpdater); !ok {
			return nil, xerrors.Errorf("not KeyUpdater, %T", op)
		} else {
			return &KeyUpdaterProcessor{
				cp:         cp,
				KeyUpdater: i,
			}, nil
		}
	}
}

func (op *KeyUpdaterProcessor) PreProcess(
	getState func(key string) (state.State, bool, error),
	_ func(valuehash.Hash, ...state.State) error,
) (state.Processor, error) {
	fact := op.Fact().(KeyUpdaterFact)

	if st, err := existsState(StateKeyAccount(fact.target), "target keys", getState); err != nil {
		return nil, err
	} else {
		op.sa = st
	}

	if ks, err := StateKeysValue(op.sa); err != nil {
		return nil, operation.NewBaseReasonErrorFromError(err)
	} else if ks.Equal(fact.Keys()) {
		return nil, operation.NewBaseReasonError("same Keys with the existing")
	}

	if st, err := existsState(StateKeyBalance(fact.target, fact.currency), "balance of target", getState); err != nil {
		return nil, err
	} else {
		op.sb = NewAmountState(st, fact.currency)
	}

	if err := checkFactSignsByState(fact.target, op.Signs(), getState); err != nil {
		return nil, operation.NewBaseReasonError("invalid signing: %w", err)
	}

	var feeer Feeer
	if i, found := op.cp.Feeer(fact.currency); !found {
		return nil, operation.NewBaseReasonError("currency, %q not found of KeyUpdater", fact.currency)
	} else {
		feeer = i
	}

	if fee, err := feeer.Fee(ZeroBig); err != nil {
		return nil, operation.NewBaseReasonErrorFromError(err)
	} else {
		switch b, err := StateBalanceValue(op.sb); {
		case err != nil:
			return nil, operation.NewBaseReasonErrorFromError(err)
		case b.Big().Compare(fee) < 0:
			return nil, operation.NewBaseReasonError("insufficient balance with fee")
		default:
			op.fee = fee
		}
	}

	return op, nil
}

func (op *KeyUpdaterProcessor) Process(
	_ func(key string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) error {
	fact := op.Fact().(KeyUpdaterFact)

	op.sb = op.sb.Sub(op.fee).AddFee(op.fee)
	if st, err := SetStateKeysValue(op.sa, fact.keys); err != nil {
		return err
	} else {
		return setState(fact.Hash(), st, op.sb)
	}
}
