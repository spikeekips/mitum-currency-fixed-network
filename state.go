package mc

import (
	"fmt"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
)

func stateKeyKeys(a Address) string {
	return fmt.Sprintf("%s:keys", a.String())
}

func stateKeyBalance(a Address) string {
	return fmt.Sprintf("%s:balance", a.String())
}

func stateAmountValue(st state.State) (Amount, error) {
	if s, ok := st.Value().Interface().(string); !ok {
		return NilAmount, xerrors.Errorf("invalid balance value found, %T", st.Value().Interface())
	} else if a, err := NewAmountFromString(s); err != nil {
		return NilAmount, xerrors.Errorf("invalid balance value found, %q : %w", s, err)
	} else {
		return a, nil
	}
}

func stateKeysValue(st state.State) (Keys, error) {
	if s, ok := st.Value().Interface().(Keys); !ok {
		return Keys{}, xerrors.Errorf("invalid Keys value found, %T", st.Value().Interface())
	} else {
		return s, nil
	}
}

func setStateAmountValue(st state.StateUpdater, v Amount) error {
	if uv, err := state.NewStringValue(v); err != nil {
		return err
	} else if err := st.SetValue(uv); err != nil {
		return err
	}

	return nil
}

func checkFactSignsByState(
	address Address,
	fs []operation.FactSign,
	getState func(key string) (state.StateUpdater, bool, error),
) error {
	var keys Keys
	switch st, found, err := getState(stateKeyKeys(address)); {
	case err != nil:
		return err
	case !found:
		return state.IgnoreOperationProcessingError.Errorf("keys for address not found")
	default:
		if ks, err := stateKeysValue(st); err != nil {
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

func loadState(
	key string,
	getState func(key string) (state.StateUpdater, bool, error),
) (state.StateUpdater, error) {
	switch st, found, err := getState(key); {
	case err != nil:
		return nil, err
	case !found:
		return nil, state.IgnoreOperationProcessingError
	default:
		return st, nil
	}
}
