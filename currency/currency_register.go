package currency

import (
	"github.com/pkg/errors"

	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	CurrencyRegisterFactType = hint.Type("mitum-currency-currency-register-operation-fact")
	CurrencyRegisterFactHint = hint.NewHint(CurrencyRegisterFactType, "v0.0.1")
	CurrencyRegisterType     = hint.Type("mitum-currency-currency-register-operation")
	CurrencyRegisterHint     = hint.NewHint(CurrencyRegisterType, "v0.0.1")
)

type CurrencyRegisterFact struct {
	h        valuehash.Hash
	token    []byte
	currency CurrencyDesign
}

func NewCurrencyRegisterFact(token []byte, de CurrencyDesign) CurrencyRegisterFact {
	fact := CurrencyRegisterFact{
		token:    token,
		currency: de,
	}

	fact.h = fact.GenerateHash()

	return fact
}

func (CurrencyRegisterFact) Hint() hint.Hint {
	return CurrencyRegisterFactHint
}

func (fact CurrencyRegisterFact) Hash() valuehash.Hash {
	return fact.h
}

func (fact CurrencyRegisterFact) Bytes() []byte {
	return util.ConcatBytesSlice(fact.token, fact.currency.Bytes())
}

func (fact CurrencyRegisterFact) IsValid([]byte) error {
	if len(fact.token) < 1 {
		return errors.Errorf("empty token for CurrencyRegisterFact")
	}

	if err := isvalid.Check([]isvalid.IsValider{
		fact.h,
		fact.currency,
	}, nil, false); err != nil {
		return errors.Wrap(err, "invalid fact")
	}

	if fact.currency.GenesisAccount() == nil {
		return errors.Errorf("empty genesis account")
	}

	if !fact.h.Equal(fact.GenerateHash()) {
		return isvalid.InvalidError.Errorf("wrong Fact hash")
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
	operation.BaseOperation
	Memo string
}

func NewCurrencyRegister(fact CurrencyRegisterFact, fs []operation.FactSign, memo string) (CurrencyRegister, error) {
	bo, err := operation.NewBaseOperationFromFact(CurrencyRegisterHint, fact, fs)
	if err != nil {
		return CurrencyRegister{}, err
	}
	op := CurrencyRegister{BaseOperation: bo, Memo: memo}

	op.BaseOperation = bo.SetHash(op.GenerateHash())

	return op, nil
}

func (CurrencyRegister) Hint() hint.Hint {
	return CurrencyRegisterHint
}

func (op CurrencyRegister) IsValid(networkID []byte) error {
	if err := IsValidMemo(op.Memo); err != nil {
		return err
	}

	return operation.IsValidOperation(op, networkID)
}
