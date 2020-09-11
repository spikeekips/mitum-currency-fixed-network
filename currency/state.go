package currency

import (
	"fmt"
	"strings"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
)

func StateKeyAccount(a base.Address) string {
	return fmt.Sprintf("%s:account", a.String())
}

func IsStateAccountKey(key string) bool {
	return strings.HasSuffix(key, ":account")
}

func StateKeyBalance(a base.Address) string {
	return fmt.Sprintf("%s:balance", a.String())
}

func StateAmountValue(st state.State) (Amount, error) {
	if i := st.Value(); i == nil {
		return ZeroAmount, nil
	} else if s, ok := i.Interface().(string); !ok {
		return NilAmount, xerrors.Errorf("invalid balance value found, %T", st.Value().Interface())
	} else if a, err := NewAmountFromString(s); err != nil {
		return NilAmount, xerrors.Errorf("invalid balance value found, %q : %w", s, err)
	} else {
		return a, nil
	}
}

func StateKeysValue(st state.State) (Keys, error) {
	if ac, err := LoadStateAccountValue(st); err != nil {
		return Keys{}, err
	} else {
		return ac.Keys(), nil
	}
}

func LoadStateAccountValue(st state.State) (Account, error) {
	v := st.Value()
	if v == nil {
		return Account{}, storage.NotFoundError.Errorf("account not found in State")
	}

	if s, ok := v.Interface().(Account); !ok {
		return Account{}, xerrors.Errorf("invalid account value found, %T", v.Interface())
	} else {
		return s, nil
	}
}

func SetStateAccountValue(st state.State, v Account) (state.State, error) {
	if uv, err := state.NewHintedValue(v); err != nil {
		return nil, err
	} else {
		return st.SetValue(uv)
	}
}

func SetStateKeysValue(st state.State, v Keys) (state.State, error) {
	var ac Account
	if a, err := LoadStateAccountValue(st); err != nil {
		if !xerrors.Is(err, storage.NotFoundError) {
			return nil, err
		}

		if n, err := NewAccountFromKeys(v); err != nil {
			return nil, err
		} else {
			ac = n
		}
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

func SetStateAmountValue(st state.State, v Amount) (state.State, error) {
	if v.Compare(ZeroAmount) < 0 {
		return nil, xerrors.Errorf("under zero Amount, %v", v)
	}

	if uv, err := state.NewStringValue(v); err != nil {
		return nil, err
	} else {
		return st.SetValue(uv)
	}
}

func checkFactSignsByState(
	address base.Address,
	fs []operation.FactSign,
	getState func(key string) (state.State, bool, error),
) error {
	var keys Keys
	if st, err := existsAccountState(StateKeyAccount(address), "keys of account", getState); err != nil {
		return err
	} else {
		if ks, err := StateKeysValue(st); err != nil {
			return state.IgnoreOperationProcessingError.Wrap(err)
		} else {
			keys = ks
		}
	}

	if err := checkThreshold(fs, keys); err != nil {
		return state.IgnoreOperationProcessingError.Wrap(err)
	}

	return nil
}

func checkExistsAccountState(
	key string,
	getState func(key string) (state.State, bool, error),
) error {
	switch _, found, err := getState(key); {
	case err != nil:
		return err
	case !found:
		return state.IgnoreOperationProcessingError.Errorf("account state, %q does not exist", key)
	default:
		return nil
	}
}

func existsAccountState(
	k,
	name string,
	getState func(key string) (state.State, bool, error),
) (state.State, error) {
	switch st, found, err := getState(k); {
	case err != nil:
		return nil, err
	case !found:
		return nil, state.IgnoreOperationProcessingError.Errorf("%s does not exist", name)
	default:
		return st, nil
	}
}

func notExistsAccountState(
	k,
	name string,
	getState func(key string) (state.State, bool, error),
) (state.State, error) {
	switch st, found, err := getState(k); {
	case err != nil:
		return nil, err
	case found:
		return nil, state.IgnoreOperationProcessingError.Errorf("%s already exists", name)
	default:
		return st, nil
	}
}
