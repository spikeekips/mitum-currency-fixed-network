package currency

import (
	"github.com/pkg/errors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	CurrencyRegisterFactType   = hint.Type("mitum-currency-currency-register-operation-fact")
	CurrencyRegisterFactHint   = hint.NewHint(CurrencyRegisterFactType, "v0.0.1")
	CurrencyRegisterFactHinter = CurrencyRegisterFact{BaseHinter: hint.NewBaseHinter(CurrencyRegisterFactHint)}
	CurrencyRegisterType       = hint.Type("mitum-currency-currency-register-operation")
	CurrencyRegisterHint       = hint.NewHint(CurrencyRegisterType, "v0.0.1")
	CurrencyRegisterHinter     = CurrencyRegister{BaseOperation: operationHinter(CurrencyRegisterHint)}
)

type CurrencyRegisterFact struct {
	hint.BaseHinter
	h        valuehash.Hash
	token    []byte
	currency CurrencyDesign
}

func NewCurrencyRegisterFact(token []byte, de CurrencyDesign) CurrencyRegisterFact {
	fact := CurrencyRegisterFact{
		BaseHinter: hint.NewBaseHinter(CurrencyRegisterFactHint),
		token:      token,
		currency:   de,
	}

	fact.h = fact.GenerateHash()

	return fact
}

func (fact CurrencyRegisterFact) Hash() valuehash.Hash {
	return fact.h
}

func (fact CurrencyRegisterFact) Bytes() []byte {
	return util.ConcatBytesSlice(fact.token, fact.currency.Bytes())
}

func (fact CurrencyRegisterFact) IsValid(b []byte) error {
	if err := IsValidOperationFact(fact, b); err != nil {
		return err
	}

	if err := isvalid.Check([]isvalid.IsValider{fact.currency}, nil, false); err != nil {
		return errors.Wrap(err, "invalid fact")
	}

	if fact.currency.GenesisAccount() == nil {
		return errors.Errorf("empty genesis account")
	}

	return nil
}

func (fact CurrencyRegisterFact) GenerateHash() valuehash.Hash {
	return valuehash.NewSHA256(fact.Bytes())
}

func (fact CurrencyRegisterFact) Token() []byte {
	return fact.token
}

func (fact CurrencyRegisterFact) Currency() CurrencyDesign {
	return fact.currency
}

type CurrencyRegister struct {
	BaseOperation
}

func NewCurrencyRegister(fact CurrencyRegisterFact, fs []base.FactSign, memo string) (CurrencyRegister, error) {
	bo, err := NewBaseOperationFromFact(CurrencyRegisterHint, fact, fs, memo)
	if err != nil {
		return CurrencyRegister{}, err
	}

	return CurrencyRegister{BaseOperation: bo}, nil
}
