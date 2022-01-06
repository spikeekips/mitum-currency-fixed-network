package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	AccountType   = hint.Type("mitum-currency-account")
	AccountHint   = hint.NewHint(AccountType, "v0.0.1")
	AccountHinter = Account{BaseHinter: hint.NewBaseHinter(AccountHint)}
)

type Account struct {
	hint.BaseHinter
	h       valuehash.Hash
	address base.Address
	keys    AccountKeys
}

func NewAccount(address base.Address, keys AccountKeys) (Account, error) {
	if err := address.IsValid(nil); err != nil {
		return Account{}, err
	}
	if keys != nil {
		if err := keys.IsValid(nil); err != nil {
			return Account{}, err
		}
	}

	ac := Account{BaseHinter: hint.NewBaseHinter(AccountHint), address: address, keys: keys}
	ac.h = ac.GenerateHash()

	return ac, nil
}

func NewAccountFromKeys(keys AccountKeys) (Account, error) {
	if a, err := NewAddressFromKeys(keys); err != nil {
		return Account{}, err
	} else if ac, err := NewAccount(a, keys); err != nil {
		return Account{}, err
	} else {
		return ac, nil
	}
}

func (ac Account) Bytes() []byte {
	bs := make([][]byte, 2)
	bs[0] = ac.address.Bytes()

	if ac.keys != nil {
		bs[1] = ac.keys.Bytes()
	}

	return util.ConcatBytesSlice(bs...)
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

func (ac Account) Keys() AccountKeys {
	return ac.keys
}

func (ac Account) SetKeys(keys AccountKeys) (Account, error) {
	if err := keys.IsValid(nil); err != nil {
		return Account{}, err
	}

	ac.keys = keys

	return ac, nil
}

func (ac Account) IsEmpty() bool {
	return ac.h == nil || ac.h.IsEmpty()
}

func ZeroAccount(cid CurrencyID) (Account, error) {
	return NewAccount(ZeroAddress(cid), nil)
}
