package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	AmountStateType = hint.MustNewType(0xa0, 0x11, "mitum-currency-amount-state")
	AmountStateHint = hint.MustHint(AmountStateType, "0.0.1")
)

type AmountState struct {
	state.State
	add Amount
	fee Amount
}

func NewAmountState(st state.State) AmountState {
	if sst, ok := st.(AmountState); ok {
		return sst
	}

	return AmountState{
		State: st,
		add:   ZeroAmount,
		fee:   ZeroAmount,
	}
}

func (st AmountState) Hint() hint.Hint {
	return AmountStateHint
}

func (st AmountState) IsValid(b []byte) error {
	if err := st.State.IsValid(b); err != nil {
		return err
	}

	if st.fee.Compare(ZeroAmount) < 0 {
		return xerrors.Errorf("invalid fee, %v", st.fee)
	}

	return nil
}

func (st AmountState) Bytes() []byte {
	return util.ConcatBytesSlice(
		st.State.Bytes(),
		st.fee.Bytes(),
	)
}

func (st AmountState) GenerateHash() valuehash.Hash {
	return valuehash.NewSHA256(st.Bytes())
}

func (st AmountState) Merge(base state.State) (state.State, error) {
	if b, err := StateAmountValue(base); err != nil {
		return nil, err
	} else {
		return SetStateAmountValue(st.AddFee(base.(AmountState).fee), b.Add(st.add))
	}
}

func (st AmountState) Add(a Amount) AmountState {
	st.add = st.add.Add(a)

	return st
}

func (st AmountState) Fee() Amount {
	return st.fee
}

func (st AmountState) AddFee(fee Amount) AmountState {
	st.fee = st.fee.Add(fee)

	return st
}

func (st AmountState) Sub(a Amount) AmountState {
	st.add = st.add.Sub(a)

	return st
}

func (st AmountState) SetValue(v state.Value) (state.State, error) {
	if s, err := st.State.SetValue(v); err != nil {
		return nil, err
	} else {
		st.State = s

		return st, nil
	}
}

func (st AmountState) SetHash(h valuehash.Hash) (state.State, error) {
	if s, err := st.State.SetHash(h); err != nil {
		return nil, err
	} else {
		st.State = s

		return st, nil
	}
}

func (st AmountState) SetHeight(h base.Height) state.State {
	st.State = st.State.SetHeight(h)

	return st
}

func (st AmountState) SetPreviousHeight(h base.Height) (state.State, error) {
	if s, err := st.State.SetPreviousHeight(h); err != nil {
		return nil, err
	} else {
		st.State = s

		return st, nil
	}
}

func (st AmountState) SetOperation(ops []valuehash.Hash) state.State {
	st.State = st.State.SetOperation(ops)

	return st
}

func (st AmountState) Clear() state.State {
	st.State = st.State.Clear()

	st.add = ZeroAmount
	st.fee = ZeroAmount

	return st
}
