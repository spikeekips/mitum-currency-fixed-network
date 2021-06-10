package digest

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/hint"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum-currency/currency"
)

var (
	AccountValueType = hint.Type("mitum-currency-account-value")
	AccountValueHint = hint.NewHint(AccountValueType, "v0.0.1")
)

type AccountValue struct {
	ac             currency.Account
	balance        []currency.Amount
	height         base.Height
	previousHeight base.Height
}

func NewAccountValue(st state.State) (AccountValue, error) {
	var ac currency.Account
	switch a, ok, err := IsAccountState(st); {
	case err != nil:
		return AccountValue{}, err
	case !ok:
		return AccountValue{}, xerrors.Errorf("not state for currency.Account, %T", st.Value().Interface())
	default:
		ac = a
	}

	return AccountValue{
		ac:             ac,
		height:         st.Height(),
		previousHeight: st.PreviousHeight(),
	}, nil
}

func (AccountValue) Hint() hint.Hint {
	return AccountValueHint
}

func (va AccountValue) Account() currency.Account {
	return va.ac
}

func (va AccountValue) Balance() []currency.Amount {
	return va.balance
}

func (va AccountValue) Height() base.Height {
	return va.height
}

func (va AccountValue) SetHeight(height base.Height) AccountValue {
	va.height = height

	return va
}

func (va AccountValue) PreviousHeight() base.Height {
	return va.previousHeight
}

func (va AccountValue) SetPreviousHeight(height base.Height) AccountValue {
	va.previousHeight = height

	return va
}

func (va AccountValue) SetBalance(balance []currency.Amount) AccountValue {
	va.balance = balance

	return va
}
