package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	AccountType = hint.Type("mitum-currency-account")
	AccountHint = hint.NewHint(AccountType, "v0.0.1")
)

type Account struct {
	h       valuehash.Hash
	address base.Address
	keys    Keys
}

func NewAccount(address base.Address, keys Keys) (Account, error) {
	if err := address.IsValid(nil); err != nil {
		return Account{}, err
	}
	if err := keys.IsValid(nil); err != nil {
		return Account{}, err
	}

	ac := Account{address: address, keys: keys}
	ac.h = ac.GenerateHash()

	return ac, nil
}

func NewAccountFromKeys(keys Keys) (Account, error) {
	if a, err := NewAddressFromKeys(keys); err != nil {
		return Account{}, err
	} else if ac, err := NewAccount(a, keys); err != nil {
		return Account{}, err
	} else {
		return ac, nil
	}
}

func (Account) Hint() hint.Hint {
	return AccountHint
}

func (ac Account) Bytes() []byte {
	return util.ConcatBytesSlice(
		ac.address.Bytes(),
		ac.keys.Bytes(),
	)
}

func (ac Account) Hash() valuehash.Hash {
	return ac.h
}

func (ac Account) GenerateHash() valuehash.Hash {
	return valuehash.NewSHA256(ac.Bytes())
}

func (ac Account) Address() base.Address {
	return ac.address
}

func (ac Account) Keys() Keys {
	return ac.keys
}

func (ac Account) SetKeys(keys Keys) (Account, error) {
	if err := keys.IsValid(nil); err != nil {
		return Account{}, err
	}

	ac.keys = keys

	return ac, nil
}

func (ac Account) IsEmpty() bool {
	return ac.h == nil || ac.h.IsEmpty()
}
