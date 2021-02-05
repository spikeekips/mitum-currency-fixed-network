package currency

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"golang.org/x/xerrors"
)

func DecodeAmount(enc encoder.Encoder, b []byte) (Amount, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return Amount{}, err
	} else if i == nil {
		return Amount{}, nil
	} else if v, ok := i.(Amount); !ok {
		return Amount{}, hint.InvalidTypeError.Errorf("not Amount; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeCreateAccountsItem(enc encoder.Encoder, b []byte) (CreateAccountsItem, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(CreateAccountsItem); !ok {
		return nil, hint.InvalidTypeError.Errorf("not CreateAccountsItem; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeTransfersItem(enc encoder.Encoder, b []byte) (TransfersItem, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(TransfersItem); !ok {
		return nil, hint.InvalidTypeError.Errorf("not TransfersItem; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeCurrencyPolicy(enc encoder.Encoder, b []byte) (CurrencyPolicy, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return CurrencyPolicy{}, err
	} else if i == nil {
		return CurrencyPolicy{}, nil
	} else if v, ok := i.(CurrencyPolicy); !ok {
		return CurrencyPolicy{}, hint.InvalidTypeError.Errorf("not CurrencyPolicy; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeFeeer(enc encoder.Encoder, b []byte) (Feeer, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if i == nil {
		return nil, nil
	} else if v, ok := i.(Feeer); !ok {
		return nil, hint.InvalidTypeError.Errorf("not Feeer; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeCurrencyDesign(enc encoder.Encoder, b []byte) (CurrencyDesign, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return CurrencyDesign{}, err
	} else if i == nil {
		return CurrencyDesign{}, nil
	} else if v, ok := i.(CurrencyDesign); !ok {
		return CurrencyDesign{}, hint.InvalidTypeError.Errorf("not CurrencyDesign; type=%T", i)
	} else {
		return v, nil
	}
}

func DecodeAccount(enc encoder.Encoder, b []byte) (Account, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return Account{}, err
	} else if ac, ok := i.(Account); !ok {
		return Account{}, xerrors.Errorf("not Account: %T", i)
	} else {
		return ac, nil
	}
}
