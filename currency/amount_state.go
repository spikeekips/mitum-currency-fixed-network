package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	AmountStateType = hint.MustNewType(0xa0, 0x23, "mitum-currency-amount-state")
	AmountStateHint = hint.MustHint(AmountStateType, "0.0.1")
)

type AmountState struct {
	state.State
	cid CurrencyID
	add Big
	fee Big
}

func NewAmountState(st state.State, cid CurrencyID) AmountState {
	if sst, ok := st.(AmountState); ok {
		return sst
	}

	return AmountState{
		State: st,
		cid:   cid,
		add:   ZeroBig,
		fee:   ZeroBig,
	}
}

func (st AmountState) Hint() hint.Hint {
	return AmountStateHint
}

func (st AmountState) IsValid(b []byte) error {
	if err := st.State.IsValid(b); err != nil {
		return err
	}

	if !st.fee.OverNil() {
		return xerrors.Errorf("invalid fee; under zero, %v", st.fee)
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
	var am Amount
	if b, err := StateBalanceValue(base); err != nil {
		if xerrors.Is(err, storage.NotFoundError) {
			am = NewZeroAmount(st.cid)
		} else {
			return nil, err
		}
	} else {
		am = b
	}

	return SetStateBalanceValue(
		st.AddFee(base.(AmountState).fee),
		am.WithBig(am.Big().Add(st.add)),
	)
}

func (st AmountState) Currency() CurrencyID {
	return st.cid
}

func (st AmountState) Fee() Big {
	return st.fee
}

func (st AmountState) AddFee(fee Big) AmountState {
	st.fee = st.fee.Add(fee)

	return st
}

func (st AmountState) Add(a Big) AmountState {
	st.add = st.add.Add(a)

	return st
}

func (st AmountState) Sub(a Big) AmountState {
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

	st.add = ZeroBig
	st.fee = ZeroBig

	return st
}
