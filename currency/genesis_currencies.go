package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	GenesisCurrenciesFactType = hint.MustNewType(0xa0, 0x20, "mitum-currency-genesis-currencies-operation-fact")
	GenesisCurrenciesFactHint = hint.MustHint(GenesisCurrenciesFactType, "0.0.1")
	GenesisCurrenciesType     = hint.MustNewType(0xa0, 0x21, "mitum-currency-genesis-currencies-operation")
	GenesisCurrenciesHint     = hint.MustHint(GenesisCurrenciesType, "0.0.1")
)

type GenesisCurrenciesFact struct {
	h              valuehash.Hash
	token          []byte
	genesisNodeKey key.Publickey
	keys           Keys
	cs             []CurrencyDesign
}

func NewGenesisCurrenciesFact(
	token []byte,
	genesisNodeKey key.Publickey,
	keys Keys,
	cs []CurrencyDesign,
) GenesisCurrenciesFact {
	fact := GenesisCurrenciesFact{
		token:          token,
		genesisNodeKey: genesisNodeKey,
		keys:           keys,
		cs:             cs,
	}

	fact.h = fact.GenerateHash()

	return fact
}

func (fact GenesisCurrenciesFact) Hint() hint.Hint {
	return GenesisCurrenciesFactHint
}

func (fact GenesisCurrenciesFact) Hash() valuehash.Hash {
	return fact.h
}

func (fact GenesisCurrenciesFact) Bytes() []byte {
	bs := make([][]byte, len(fact.cs)+3)
	bs[0] = fact.token
	bs[1] = []byte(fact.genesisNodeKey.String())
	bs[2] = fact.keys.Bytes()

	for i := range fact.cs {
		bs[i+3] = fact.cs[i].Bytes()
	}

	return util.ConcatBytesSlice(bs...)
}

func (fact GenesisCurrenciesFact) IsValid([]byte) error {
	if len(fact.token) < 1 {
		return xerrors.Errorf("empty token for GenesisCurrenciesFact")
	} else if len(fact.cs) < 1 {
		return xerrors.Errorf("empty GenesisCurrency for GenesisCurrenciesFact")
	}

	if err := isvalid.Check([]isvalid.IsValider{
		fact.h,
		fact.genesisNodeKey,
		fact.keys,
	}, nil, false); err != nil {
		return xerrors.Errorf("invalid fact: %w", err)
	}

	founds := map[CurrencyID]struct{}{}
	for i := range fact.cs {
		c := fact.cs[i]
		if err := c.IsValid(nil); err != nil {
			return err
		} else if _, found := founds[c.Currency()]; found {
			return xerrors.Errorf("duplicated currency id found, %q", c.Currency())
		} else {
			founds[c.Currency()] = struct{}{}
		}
	}

	if !fact.h.Equal(fact.GenerateHash()) {
		return isvalid.InvalidError.Errorf("wrong Fact hash")
	}

	return nil
}

func (fact GenesisCurrenciesFact) GenerateHash() valuehash.Hash {
	return valuehash.NewSHA256(fact.Bytes())
}

func (fact GenesisCurrenciesFact) Token() []byte {
	return fact.token
}

func (fact GenesisCurrenciesFact) GenesisNodeKey() key.Publickey {
	return fact.genesisNodeKey
}

func (fact GenesisCurrenciesFact) Keys() Keys {
	return fact.keys
}

func (fact GenesisCurrenciesFact) Address() (base.Address, error) {
	return NewAddressFromKeys(fact.keys)
}

func (fact GenesisCurrenciesFact) Currencies() []CurrencyDesign {
	return fact.cs
}

type GenesisCurrencies struct {
	operation.BaseOperation
}

func NewGenesisCurrencies(
	genesisNodeKey key.Privatekey,
	keys Keys,
	cs []CurrencyDesign,
	networkID base.NetworkID,
) (GenesisCurrencies, error) {
	fact := NewGenesisCurrenciesFact(networkID, genesisNodeKey.Publickey(), keys, cs)

	var fs []operation.FactSign
	if sig, err := operation.NewFactSignature(genesisNodeKey, fact, networkID); err != nil {
		return GenesisCurrencies{}, err
	} else {
		fs = []operation.FactSign{operation.NewBaseFactSign(genesisNodeKey.Publickey(), sig)}
	}

	if bo, err := operation.NewBaseOperationFromFact(GenesisCurrenciesHint, fact, fs); err != nil {
		return GenesisCurrencies{}, err
	} else {
		return GenesisCurrencies{BaseOperation: bo}, nil
	}
}

func (op GenesisCurrencies) Hint() hint.Hint {
	return GenesisCurrenciesHint
}

func (op GenesisCurrencies) IsValid(networkID []byte) error {
	if err := operation.IsValidOperation(op, networkID); err != nil {
		return err
	}

	if len(op.Signs()) != 1 {
		return xerrors.Errorf("genesis currencies should be signed only by genesis node key")
	}

	fact := op.Fact().(GenesisCurrenciesFact)
	if !fact.genesisNodeKey.Equal(op.Signs()[0].Signer()) {
		return xerrors.Errorf("not signed by genesis node key")
	}

	return nil
}
