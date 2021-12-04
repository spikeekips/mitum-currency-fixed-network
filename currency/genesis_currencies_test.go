package currency

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/stretchr/testify/suite"
)

type testGenesisCurrencies struct {
	suite.Suite
}

func (t *testGenesisCurrencies) TestNew() {
	nodeKey := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()
	rkey, err := NewBaseAccountKey(rpk.Publickey(), 100)
	t.NoError(err)
	keys, _ := NewBaseAccountKeys([]AccountKey{rkey}, 100)
	networkID := util.UUID().Bytes()

	cs := []CurrencyDesign{
		NewCurrencyDesign(MustNewAmount(NewBig(33), CurrencyID("ABC")), NewTestAddress(), NewCurrencyPolicy(ZeroBig, NewNilFeeer())),
		NewCurrencyDesign(MustNewAmount(NewBig(44), CurrencyID("DEF")), NewTestAddress(), NewCurrencyPolicy(ZeroBig, NewNilFeeer())),
	}

	gc, err := NewGenesisCurrencies(nodeKey, keys, cs, networkID)
	t.NoError(err)

	t.NoError(gc.IsValid(networkID))

	t.Implements((*base.Fact)(nil), gc.Fact())
	t.Implements((*operation.Operation)(nil), gc)

	fact := gc.Fact().(GenesisCurrenciesFact)

	t.Equal(len(cs), len(fact.cs))
	t.True(nodeKey.Publickey().Equal(fact.genesisNodeKey))
	t.True(keys.Equal(fact.keys))
	t.Equal(networkID, fact.token)
	t.Equal(cs, fact.cs)
}

func (t *testGenesisCurrencies) TestDuplicatedCurrencyID() {
	nodeKey := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()
	rkey, err := NewBaseAccountKey(rpk.Publickey(), 100)
	t.NoError(err)
	keys, _ := NewBaseAccountKeys([]AccountKey{rkey}, 100)
	networkID := util.UUID().Bytes()

	cs := []CurrencyDesign{
		NewCurrencyDesign(MustNewAmount(NewBig(33), CurrencyID("ABC")), NewTestAddress(), NewCurrencyPolicy(ZeroBig, NewNilFeeer())),
		NewCurrencyDesign(MustNewAmount(NewBig(44), CurrencyID("ABC")), NewTestAddress(), NewCurrencyPolicy(ZeroBig, NewNilFeeer())),
	}

	gc, err := NewGenesisCurrencies(nodeKey, keys, cs, networkID)
	t.NoError(err)

	err = gc.IsValid(networkID)
	t.Contains(err.Error(), "duplicated currency id found")
}

func (t *testGenesisCurrencies) TestEmptyGenesisCurrencies() {
	nodeKey := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()
	rkey, err := NewBaseAccountKey(rpk.Publickey(), 100)
	t.NoError(err)
	keys, _ := NewBaseAccountKeys([]AccountKey{rkey}, 100)
	networkID := util.UUID().Bytes()

	gc, err := NewGenesisCurrencies(nodeKey, keys, nil, networkID)
	t.NoError(err)

	err = gc.IsValid(networkID)
	t.Contains(err.Error(), "empty GenesisCurrency")
}

func TestGenesisCurrencies(t *testing.T) {
	suite.Run(t, new(testGenesisCurrencies))
}

func testGenesisCurrenciesEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestOperationEncode)

	t.enc = enc
	t.newObject = func() interface{} {
		nodeKey := key.MustNewBTCPrivatekey()
		rpk := key.MustNewBTCPrivatekey()
		rkey, err := NewBaseAccountKey(rpk.Publickey(), 100)
		t.NoError(err)
		keys, _ := NewBaseAccountKeys([]AccountKey{rkey}, 100)
		networkID := util.UUID().Bytes()

		cs := []CurrencyDesign{
			NewCurrencyDesign(MustNewAmount(NewBig(33), CurrencyID("ABC")), NewTestAddress(), NewCurrencyPolicy(ZeroBig, NewNilFeeer())),
			NewCurrencyDesign(MustNewAmount(NewBig(44), CurrencyID("DEF")), NewTestAddress(), NewCurrencyPolicy(ZeroBig, NewNilFeeer())),
		}

		gc, err := NewGenesisCurrencies(nodeKey, keys, cs, networkID)
		t.NoError(err)

		return gc
	}

	t.compare = func(a, b interface{}) {
		ca := a.(GenesisCurrencies)
		cb := b.(GenesisCurrencies)
		fact := ca.Fact().(GenesisCurrenciesFact)
		ufact := cb.Fact().(GenesisCurrenciesFact)

		t.True(fact.genesisNodeKey.Equal(ufact.genesisNodeKey))

		t.True(fact.keys.Hash().Equal(ufact.keys.Hash()))
		for i := range fact.keys.Keys() {
			t.Equal(fact.keys.Keys()[i].Bytes(), ufact.keys.Keys()[i].Bytes())
		}
		t.Equal(fact.keys.Threshold(), ufact.keys.Threshold())

		t.Equal(len(fact.cs), len(ufact.cs))
		for i := range fact.cs {
			t.True(fact.cs[i].Equal(ufact.cs[i].Amount))
		}
	}

	return t
}

func TestGenesisCurrenciesEncodeJSON(t *testing.T) {
	suite.Run(t, testGenesisCurrenciesEncode(jsonenc.NewEncoder()))
}

func TestGenesisCurrenciesEncodeBSON(t *testing.T) {
	suite.Run(t, testGenesisCurrenciesEncode(bsonenc.NewEncoder()))
}
