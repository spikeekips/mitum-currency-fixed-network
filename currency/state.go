package currency

import (
	"fmt"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
)

type AmountState struct {
	state.State
	add    Amount
	exists bool
}

func NewAmountState(st state.State, exists bool) AmountState {
	return AmountState{State: st, add: ZeroAmount, exists: exists}
}

func (am AmountState) Amount() (Amount, error) {
	return StateAmountValue(am)
}

func (am AmountState) Add(a Amount) AmountState {
	am.add = am.add.Add(a)

	return am
}

func (am AmountState) Sub(a Amount) AmountState {
	am.add = am.add.Sub(a)

	return am
}

func (am AmountState) Merge(source state.State) (state.State, error) {
	if base, err := StateAmountValue(source); err != nil {
		return nil, err
	} else {
		return SetStateAmountValue(am.State, base.Add(am.add))
	}
}

func StateKeyKeys(a base.Address) string {
	return fmt.Sprintf("%s:keys", a.String())
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
	if s, ok := st.Value().Interface().(Keys); !ok {
		return Keys{}, xerrors.Errorf("invalid Keys value found, %T", st.Value().Interface())
	} else {
		return s, nil
	}
}

func SetStateKeysValue(st state.State, v Keys) (state.State, error) {
	if uv, err := state.NewHintedValue(v); err != nil {
		return nil, err
	} else {
		return st.SetValue(uv)
	}
}

func SetStateAmountValue(st state.State, v Amount) (state.State, error) {
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
	if st, err := existsAccountState(StateKeyKeys(address), "keys of account", getState); err != nil {
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
