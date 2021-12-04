package currency

import (
	"github.com/pkg/errors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	GenesisCurrenciesFactType   = hint.Type("mitum-currency-genesis-currencies-operation-fact")
	GenesisCurrenciesFactHint   = hint.NewHint(GenesisCurrenciesFactType, "v0.0.1")
	GenesisCurrenciesFactHinter = GenesisCurrenciesFact{BaseHinter: hint.NewBaseHinter(GenesisCurrenciesFactHint)}
	GenesisCurrenciesType       = hint.Type("mitum-currency-genesis-currencies-operation")
	GenesisCurrenciesHint       = hint.NewHint(GenesisCurrenciesType, "v0.0.1")
	GenesisCurrenciesHinter     = GenesisCurrencies{BaseOperation: operation.EmptyBaseOperation(GenesisCurrenciesHint)}
)

type GenesisCurrenciesFact struct {
	hint.BaseHinter
	h              valuehash.Hash
	token          []byte
	genesisNodeKey key.Publickey
	keys           AccountKeys
	cs             []CurrencyDesign
}

func NewGenesisCurrenciesFact(
	token []byte,
	genesisNodeKey key.Publickey,
	keys AccountKeys,
	cs []CurrencyDesign,
) GenesisCurrenciesFact {
	fact := GenesisCurrenciesFact{
		BaseHinter:     hint.NewBaseHinter(GenesisCurrenciesFactHint),
		token:          token,
		genesisNodeKey: genesisNodeKey,
		keys:           keys,
		cs:             cs,
	}

	fact.h = fact.GenerateHash()

	return fact
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

func (fact GenesisCurrenciesFact) IsValid(b []byte) error {
	if err := IsValidOperationFact(fact, b); err != nil {
		return err
	}

	if len(fact.cs) < 1 {
		return errors.Errorf("empty GenesisCurrency for GenesisCurrenciesFact")
	}

	if err := isvalid.Check([]isvalid.IsValider{
		fact.genesisNodeKey,
		fact.keys,
	}, nil, false); err != nil {
		return errors.Wrap(err, "invalid fact")
	}

	founds := map[CurrencyID]struct{}{}
	for i := range fact.cs {
		c := fact.cs[i]
		if err := c.IsValid(nil); err != nil {
			return err
		} else if _, found := founds[c.Currency()]; found {
			return errors.Errorf("duplicated currency id found, %q", c.Currency())
		} else {
			founds[c.Currency()] = struct{}{}
		}
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

func (fact GenesisCurrenciesFact) Keys() AccountKeys {
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
	keys AccountKeys,
	cs []CurrencyDesign,
	networkID base.NetworkID,
) (GenesisCurrencies, error) {
	fact := NewGenesisCurrenciesFact(networkID, genesisNodeKey.Publickey(), keys, cs)

	sig, err := base.NewFactSignature(genesisNodeKey, fact, networkID)
	if err != nil {
		return GenesisCurrencies{}, err
	}
	fs := []base.FactSign{base.NewBaseFactSign(genesisNodeKey.Publickey(), sig)}

	bo, err := operation.NewBaseOperationFromFact(GenesisCurrenciesHint, fact, fs)
	if err != nil {
		return GenesisCurrencies{}, err
	}
	return GenesisCurrencies{BaseOperation: bo}, nil
}

func (op GenesisCurrencies) IsValid(networkID []byte) error {
	if err := operation.IsValidOperation(op, networkID); err != nil {
		return err
	}

	if len(op.Signs()) != 1 {
		return errors.Errorf("genesis currencies should be signed only by genesis node key")
	}

	fact := op.Fact().(GenesisCurrenciesFact)
	if !fact.genesisNodeKey.Equal(op.Signs()[0].Signer()) {
		return errors.Errorf("not signed by genesis node key")
	}

	return nil
}
