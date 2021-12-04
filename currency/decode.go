package currency

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

func DecodeCurrencyPolicy(b []byte, enc encoder.Encoder) (CurrencyPolicy, error) {
	if i, err := enc.Decode(b); err != nil {
		return CurrencyPolicy{}, err
	} else if i == nil {
		return CurrencyPolicy{}, nil
	} else if v, ok := i.(CurrencyPolicy); !ok {
		return CurrencyPolicy{}, util.WrongTypeError.Errorf("not CurrencyPolicy; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeFeeer(b []byte, enc encoder.Encoder) (Feeer, error) {
	if i, err := enc.Decode(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Feeer); !ok {
		return nil, util.WrongTypeError.Errorf("not Feeer; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeCurrencyDesign(b []byte, enc encoder.Encoder) (CurrencyDesign, error) {
	if i, err := enc.Decode(b); err != nil {
		return CurrencyDesign{}, err
	} else if i == nil {
		return CurrencyDesign{}, nil
	} else if v, ok := i.(CurrencyDesign); !ok {
		return CurrencyDesign{}, util.WrongTypeError.Errorf("not CurrencyDesign; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeAccount(b []byte, enc encoder.Encoder) (Account, error) {
	if i, err := enc.Decode(b); err != nil {
		return Account{}, err
	} else if ac, ok := i.(Account); !ok {
		return Account{}, errors.Errorf("not Account: %T", i)
	} else {
		return ac, nil
	}
}

func DecodeAccountKeys(b []byte, enc encoder.Encoder) (AccountKeys, error) {
	i, err := enc.Decode(b)
	switch {
	case err != nil:
		return nil, err
	case i == nil:
		return nil, nil
	}

	v, ok := i.(AccountKeys)
	if !ok {
		return nil, util.WrongTypeError.Errorf("not AccountKeys; type=%T", i)
	}

	return v, nil
}
