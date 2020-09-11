package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

var (
	KeyUpdaterFactType = hint.MustNewType(0xa0, 0x09, "mitum-currency-keyupdater-operation-fact")
	KeyUpdaterFactHint = hint.MustHint(KeyUpdaterFactType, "0.0.1")
	KeyUpdaterType     = hint.MustNewType(0xa0, 0x10, "mitum-currency-keyupdater-operation")
	KeyUpdaterHint     = hint.MustHint(KeyUpdaterType, "0.0.1")
)

type KeyUpdaterFact struct {
	h      valuehash.Hash
	token  []byte
	target base.Address
	keys   Keys
}

func NewKeyUpdaterFact(token []byte, target base.Address, keys Keys) KeyUpdaterFact {
	ft := KeyUpdaterFact{
		token:  token,
		target: target,
		keys:   keys,
	}
	ft.h = valuehash.NewSHA256(ft.Bytes())

	return ft
}

func (ft KeyUpdaterFact) Hint() hint.Hint {
	return KeyUpdaterFactHint
}

func (ft KeyUpdaterFact) Hash() valuehash.Hash {
	return ft.h
}

func (ft KeyUpdaterFact) Bytes() []byte {
	return util.ConcatBytesSlice(
		ft.token,
		ft.target.Bytes(),
		ft.keys.Bytes(),
	)
}

func (ft KeyUpdaterFact) IsValid([]byte) error {
	if len(ft.token) < 1 {
		return xerrors.Errorf("empty token for KeyUpdaterFact")
	}

	if err := isvalid.Check([]isvalid.IsValider{
		ft.h,
		ft.target,
		ft.keys,
	}, nil, false); err != nil {
		return err
	}

	return nil
}

func (ft KeyUpdaterFact) Token() []byte {
	return ft.token
}

func (ft KeyUpdaterFact) Target() base.Address {
	return ft.target
}

func (ft KeyUpdaterFact) Keys() Keys {
	return ft.keys
}

type KeyUpdater struct {
	operation.BaseOperation
	Memo string
}

func NewKeyUpdater(fact KeyUpdaterFact, fs []operation.FactSign, memo string) (KeyUpdater, error) {
	if bo, err := operation.NewBaseOperationFromFact(KeyUpdaterHint, fact, fs); err != nil {
		return KeyUpdater{}, err
	} else {
		op := KeyUpdater{BaseOperation: bo, Memo: memo}

		op.BaseOperation = bo.SetHash(op.GenerateHash())

		return op, nil
	}
}

func (op KeyUpdater) Hint() hint.Hint {
	return KeyUpdaterHint
}

func (op KeyUpdater) IsValid(networkID []byte) error {
	if err := IsValidMemo(op.Memo); err != nil {
		return err
	}

	return operation.IsValidOperation(op, networkID)
}

func (op KeyUpdater) GenerateHash() valuehash.Hash {
	bs := make([][]byte, len(op.Signs())+1)
	for i := range op.Signs() {
		bs[i] = op.Signs()[i].Bytes()
	}

	bs[len(bs)-1] = []byte(op.Memo)

	e := util.ConcatBytesSlice(op.Fact().Hash().Bytes(), util.ConcatBytesSlice(bs...))

	return valuehash.NewSHA256(e)
}

func (op KeyUpdater) AddFactSigns(fs ...operation.FactSign) (operation.FactSignUpdater, error) {
	if o, err := op.BaseOperation.AddFactSigns(fs...); err != nil {
		return nil, err
	} else {
		op.BaseOperation = o.(operation.BaseOperation)
	}

	op.BaseOperation = op.SetHash(op.GenerateHash())

	return op, nil
}

func (op KeyUpdater) Process(
	func(key string) (state.State, bool, error),
	func(valuehash.Hash, ...state.State) error,
) error {
	return nil
}

type KeyUpdaterProcessor struct {
	KeyUpdater
	fa  FeeAmount
	sa  state.State
	sb  AmountState
	fee Amount
}

func (op *KeyUpdaterProcessor) PreProcess(
	getState func(key string) (state.State, bool, error),
	_ func(valuehash.Hash, ...state.State) error,
) (state.Processor, error) {
	fact := op.Fact().(KeyUpdaterFact)

	if st, err := existsAccountState(StateKeyAccount(fact.target), "target keys", getState); err != nil {
		return nil, err
	} else {
		op.sa = st
	}

	if st, err := existsAccountState(StateKeyBalance(fact.target), "balance of target", getState); err != nil {
		return nil, err
	} else {
		op.sb = NewAmountState(st)
	}

	if err := checkFactSignsByState(fact.target, op.Signs(), getState); err != nil {
		return nil, state.IgnoreOperationProcessingError.Errorf("invalid signing: %w", err)
	}

	if ks, err := StateKeysValue(op.sa); err != nil {
		return nil, state.IgnoreOperationProcessingError.Wrap(err)
	} else if ks.Equal(fact.Keys()) {
		return nil, state.IgnoreOperationProcessingError.Errorf("same Keys with the existing")
	}

	if fee, err := op.fa.Fee(ZeroAmount); err != nil {
		return nil, state.IgnoreOperationProcessingError.Wrap(err)
	} else {
		switch b, err := StateAmountValue(op.sb); {
		case err != nil:
			return nil, state.IgnoreOperationProcessingError.Wrap(err)
		case b.Compare(fee) < 0:
			return nil, state.IgnoreOperationProcessingError.Errorf("insufficient balance with fee")
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
