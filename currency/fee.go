package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	FeeOperationFactType = hint.MustNewType(0xa0, 0x12, "mitum-currency-fee-operation-fact")
	FeeOperationFactHint = hint.MustHint(FeeOperationFactType, "0.0.1")
	FeeOperationType     = hint.MustNewType(0xa0, 0x13, "mitum-currency-fee-operation")
	FeeOperationHint     = hint.MustHint(FeeOperationType, "0.0.1")
)

type FeeOperationFact struct {
	h        valuehash.Hash
	token    []byte
	fa       string
	receiver base.Address
	fee      Amount
}

func NewFeeOperationFact(feeAmount FeeAmount, height base.Height, receiver base.Address, sum Amount) FeeOperationFact {
	ft := FeeOperationFact{
		token:    height.Bytes(),
		fa:       feeAmount.Verbose(),
		receiver: receiver,
		fee:      sum,
	}
	ft.h = valuehash.NewSHA256(ft.Bytes())

	return ft
}

func (ft FeeOperationFact) Hint() hint.Hint {
	return FeeOperationFactHint
}

func (ft FeeOperationFact) Hash() valuehash.Hash {
	return ft.h
}

func (ft FeeOperationFact) Bytes() []byte {
	return util.ConcatBytesSlice(
		ft.token,
		ft.receiver.Bytes(),
		ft.fee.Bytes(),
	)
}

func (ft FeeOperationFact) IsValid([]byte) error {
	if len(ft.token) < 1 {
		return xerrors.Errorf("empty token for FeeOperationFact")
	}

	if err := isvalid.Check([]isvalid.IsValider{
		ft.h,
		ft.receiver,
		ft.fee,
	}, nil, false); err != nil {
		return err
	}

	return nil
}

func (ft FeeOperationFact) Token() []byte {
	return ft.token
}

func (ft FeeOperationFact) Receiver() base.Address {
	return ft.receiver
}

func (ft FeeOperationFact) Fee() Amount {
	return ft.fee
}

type FeeOperation struct {
	fact FeeOperationFact
	h    valuehash.Hash
}

func NewFeeOperation(fact FeeOperationFact) FeeOperation {
	op := FeeOperation{fact: fact}
	op.h = op.GenerateHash()

	return op
}

func (op FeeOperation) Hint() hint.Hint {
	return FeeOperationHint
}

func (op FeeOperation) Fact() base.Fact {
	return op.fact
}

func (op FeeOperation) Hash() valuehash.Hash {
	return op.h
}

func (op FeeOperation) Signs() []operation.FactSign {
	return nil
}

func (op FeeOperation) IsValid([]byte) error {
	if err := op.Hint().IsValid(nil); err != nil {
		return err
	}

	if l := len(op.fact.Token()); l < 1 {
		return isvalid.InvalidError.Errorf("FeeOperation has empty token")
	} else if l > operation.MaxTokenSize {
		return isvalid.InvalidError.Errorf("FeeOperation token size too large: %d > %d", l, operation.MaxTokenSize)
	}

	if err := op.Fact().IsValid(nil); err != nil {
		return err
	}

	if !op.Hash().Equal(op.GenerateHash()) {
		return isvalid.InvalidError.Errorf("wrong FeeOperation hash")
	}

	return nil
}

func (op FeeOperation) GenerateHash() valuehash.Hash {
	return valuehash.NewSHA256(op.Fact().Hash().Bytes())
}

func (op FeeOperation) AddFactSigns(...operation.FactSign) (operation.FactSignUpdater, error) {
	return nil, nil
}

func (op FeeOperation) Process(
	getState func(key string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) error {
	fact := op.Fact().(FeeOperationFact)
	if err := checkExistsAccountState(StateKeyAccount(fact.receiver), getState); err != nil {
		return err
	}

	var sb state.State
	if st, err := existsAccountState(StateKeyBalance(fact.receiver), "balance of receiver", getState); err != nil {
		return err
	} else {
		sb = st
	}

	if b, err := StateAmountValue(sb); err != nil {
		return state.IgnoreOperationProcessingError.Wrap(err)
	} else {
		if st, err := SetStateAmountValue(sb, b.Add(fact.Fee())); err != nil {
			return xerrors.Errorf("failed to add fee: %w", err)
		} else {
			sb = st
		}
	}

	return setState(fact.Hash(), sb)
}
