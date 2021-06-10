package currency

import (
	"fmt"
	"strings"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
)

var (
	StateKeyAccountSuffix        = ":account"
	StateKeyBalanceSuffix        = ":balance"
	StateKeyCurrencyDesignPrefix = "currencydesign:"
)

func StateAddressKeyPrefix(a base.Address) string {
	return fmt.Sprintf("%s-%s", a.Raw(), a.Hint().Type())
}

func StateBalanceKeyPrefix(a base.Address, cid CurrencyID) string {
	return fmt.Sprintf("%s-%s", StateAddressKeyPrefix(a), cid)
}

func StateKeyAccount(a base.Address) string {
	return fmt.Sprintf("%s%s", StateAddressKeyPrefix(a), StateKeyAccountSuffix)
}

func IsStateAccountKey(key string) bool {
	return strings.HasSuffix(key, StateKeyAccountSuffix)
}

func LoadStateAccountValue(st state.State) (Account, error) {
	v := st.Value()
	if v == nil {
		return Account{}, util.NotFoundError.Errorf("account not found in State")
	}

	s, ok := v.Interface().(Account)
	if !ok {
		return Account{}, xerrors.Errorf("invalid account value found, %T", v.Interface())
	}
	return s, nil
}

func SetStateAccountValue(st state.State, v Account) (state.State, error) {
	uv, err := state.NewHintedValue(v)
	if err != nil {
		return nil, err
	}
	return st.SetValue(uv)
}

func StateKeysValue(st state.State) (Keys, error) {
	ac, err := LoadStateAccountValue(st)
	if err != nil {
		return Keys{}, err
	}
	return ac.Keys(), nil
}

func SetStateKeysValue(st state.State, v Keys) (state.State, error) {
	var ac Account
	if a, err := LoadStateAccountValue(st); err != nil {
		if !xerrors.Is(err, util.NotFoundError) {
			return nil, err
		}

		n, err := NewAccountFromKeys(v)
		if err != nil {
			return nil, err
		}
		ac = n
	} else {
		ac = a
	}

	if uac, err := ac.SetKeys(v); err != nil {
		return nil, err
	} else if uv, err := state.NewHintedValue(uac); err != nil {
		return nil, err
	} else {
		return st.SetValue(uv)
	}
}

func StateKeyBalance(a base.Address, cid CurrencyID) string {
	return fmt.Sprintf("%s%s", StateBalanceKeyPrefix(a, cid), StateKeyBalanceSuffix)
}

func IsStateBalanceKey(key string) bool {
	return strings.HasSuffix(key, StateKeyBalanceSuffix)
}

func StateBalanceValue(st state.State) (Amount, error) {
	v := st.Value()
	if v == nil {
		return Amount{}, util.NotFoundError.Errorf("balance not found in State")
	}

	s, ok := v.Interface().(Amount)
	if !ok {
		return Amount{}, xerrors.Errorf("invalid balance value found, %T", v.Interface())
	}
	return s, nil
}

func SetStateBalanceValue(st state.State, v Amount) (state.State, error) {
	uv, err := state.NewHintedValue(v)
	if err != nil {
		return nil, err
	}
	return st.SetValue(uv)
}

func IsStateCurrencyDesignKey(key string) bool {
	return strings.HasPrefix(key, StateKeyCurrencyDesignPrefix)
}

func StateKeyCurrencyDesign(cid CurrencyID) string {
	return fmt.Sprintf("%s%s", StateKeyCurrencyDesignPrefix, cid)
}

func StateCurrencyDesignValue(st state.State) (CurrencyDesign, error) {
	v := st.Value()
	if v == nil {
		return CurrencyDesign{}, util.NotFoundError.Errorf("currency design not found in State")
	}

	s, ok := v.Interface().(CurrencyDesign)
	if !ok {
		return CurrencyDesign{}, xerrors.Errorf("invalid currency design value found, %T", v.Interface())
	}
	return s, nil
}

func SetStateCurrencyDesignValue(st state.State, v CurrencyDesign) (state.State, error) {
	uv, err := state.NewHintedValue(v)
	if err != nil {
		return nil, err
	}
	return st.SetValue(uv)
}

func checkExistsState(
	key string,
	getState func(key string) (state.State, bool, error),
) error {
	switch _, found, err := getState(key); {
	case err != nil:
		return err
	case !found:
		return operation.NewBaseReasonError("state, %q does not exist", key)
	default:
		return nil
	}
}

func existsState(
	k,
	name string,
	getState func(key string) (state.State, bool, error),
) (state.State, error) {
	switch st, found, err := getState(k); {
	case err != nil:
		return nil, err
	case !found:
		return nil, operation.NewBaseReasonError("%s does not exist", name)
	default:
		return st, nil
	}
}

func notExistsState(
	k,
	name string,
	getState func(key string) (state.State, bool, error),
) (state.State, error) {
	switch st, found, err := getState(k); {
	case err != nil:
		return nil, err
	case found:
		return nil, operation.NewBaseReasonError("%s already exists", name)
	default:
		return st, nil
	}
}
