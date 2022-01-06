package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	KeyUpdaterFactType   = hint.Type("mitum-currency-keyupdater-operation-fact")
	KeyUpdaterFactHint   = hint.NewHint(KeyUpdaterFactType, "v0.0.1")
	KeyUpdaterFactHinter = KeyUpdaterFact{BaseHinter: hint.NewBaseHinter(KeyUpdaterFactHint)}
	KeyUpdaterType       = hint.Type("mitum-currency-keyupdater-operation")
	KeyUpdaterHint       = hint.NewHint(KeyUpdaterType, "v0.0.1")
	KeyUpdaterHinter     = KeyUpdater{BaseOperation: operationHinter(KeyUpdaterHint)}
)

type KeyUpdaterFact struct {
	hint.BaseHinter
	h        valuehash.Hash
	token    []byte
	target   base.Address
	keys     AccountKeys
	currency CurrencyID
}

func NewKeyUpdaterFact(token []byte, target base.Address, keys AccountKeys, currency CurrencyID) KeyUpdaterFact {
	fact := KeyUpdaterFact{
		BaseHinter: hint.NewBaseHinter(KeyUpdaterFactHint),
		token:      token,
		target:     target,
		keys:       keys,
		currency:   currency,
	}
	fact.h = fact.GenerateHash()

	return fact
}

func (fact KeyUpdaterFact) Hash() valuehash.Hash {
	return fact.h
}

func (fact KeyUpdaterFact) GenerateHash() valuehash.Hash {
	return valuehash.NewSHA256(fact.Bytes())
}

func (fact KeyUpdaterFact) Bytes() []byte {
	return util.ConcatBytesSlice(
		fact.token,
		fact.target.Bytes(),
		fact.keys.Bytes(),
		fact.currency.Bytes(),
	)
}

func (fact KeyUpdaterFact) IsValid(b []byte) error {
	if err := IsValidOperationFact(fact, b); err != nil {
		return err
	}

	return isvalid.Check(nil, false,
		fact.target,
		fact.keys,
		fact.currency,
	)
}

func (fact KeyUpdaterFact) Token() []byte {
	return fact.token
}

func (fact KeyUpdaterFact) Target() base.Address {
	return fact.target
}

func (fact KeyUpdaterFact) Keys() AccountKeys {
	return fact.keys
}

func (fact KeyUpdaterFact) Currency() CurrencyID {
	return fact.currency
}

func (fact KeyUpdaterFact) Addresses() ([]base.Address, error) {
	return []base.Address{fact.target}, nil
}

type KeyUpdater struct {
	BaseOperation
}

func NewKeyUpdater(fact KeyUpdaterFact, fs []base.FactSign, memo string) (KeyUpdater, error) {
	bo, err := NewBaseOperationFromFact(KeyUpdaterHint, fact, fs, memo)
	if err != nil {
		return KeyUpdater{}, err
	}

	return KeyUpdater{BaseOperation: bo}, nil
}
