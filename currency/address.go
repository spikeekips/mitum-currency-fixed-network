package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	AddressType       = hint.Type("mca")
	AddressHint       = hint.NewHint(AddressType, "v0.0.1")
	AddressHinter     = Address{StringAddress: base.NewStringAddressWithHint(AddressHint, "")}
	ZeroAddressSuffix = "-X"
)

type Address struct {
	base.StringAddress
}

func NewAddress(s string) Address {
	ca := Address{StringAddress: base.NewStringAddressWithHint(AddressHint, s)}

	return ca
}

func NewAddressFromKeys(keys AccountKeys) (Address, error) {
	if err := keys.IsValid(nil); err != nil {
		return Address{}, err
	}

	return NewAddress(keys.Hash().String()), nil
}

func (ca Address) IsValid([]byte) error {
	if err := ca.StringAddress.IsValid(nil); err != nil {
		return isvalid.InvalidError.Errorf("invalid mitum currency address: %w", err)
	}

	return nil
}

func (ca Address) SetHint(ht hint.Hint) hint.Hinter {
	ca.StringAddress = ca.StringAddress.SetHint(ht).(base.StringAddress)

	return ca
}

type Addresses interface {
	Addresses() ([]base.Address, error)
}

func ZeroAddress(cid CurrencyID) Address {
	return NewAddress(cid.String() + ZeroAddressSuffix)
}
