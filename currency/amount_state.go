package currency

import (
	"github.com/pkg/errors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	amountStateType = hint.Type("mitum-currency-amount-state")
	amountStateHint = hint.NewHint(amountStateType, "v0.0.1")
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

func (AmountState) Hint() hint.Hint {
	return amountStateHint
}

func (st AmountState) IsValid(b []byte) error {
	if err := isvalid.Check(b, false, st.State); err != nil {
		return err
	}

	if !st.fee.OverNil() {
		return isvalid.InvalidError.Errorf("invalid fee; under zero, %v", st.fee)
	}

	return nil
}

func (st AmountState) Merge(b state.State) (state.State, error) {
	var am Amount
	if b, err := StateBalanceValue(b); err != nil {
		if !errors.Is(err, util.NotFoundError) {
			return nil, err
		}
		am = NewZeroAmount(st.cid)
	} else {
		am = b
	}

	return SetStateBalanceValue(
		st.AddFee(b.(AmountState).fee),
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
	s, err := st.State.SetValue(v)
	if err != nil {
		return nil, err
	}
	st.State = s

	return st, nil
}

func (st AmountState) SetHash(h valuehash.Hash) (state.State, error) {
	s, err := st.State.SetHash(h)
	if err != nil {
		return nil, err
	}
	st.State = s

	return st, nil
}

func (st AmountState) SetHeight(h base.Height) state.State {
	st.State = st.State.SetHeight(h)

	return st
}

func (st AmountState) SetPreviousHeight(h base.Height) (state.State, error) {
	s, err := st.State.SetPreviousHeight(h)
	if err != nil {
		return nil, err
	}
	st.State = s

	return st, nil
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
