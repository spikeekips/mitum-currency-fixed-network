package currency

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type testGenesisAccount struct {
	suite.Suite
}

func (t *testGenesisAccount) TestNew() {
	nodeKey := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()
	rkey, err := NewKey(rpk.Publickey(), 100)
	t.NoError(err)
	keys, _ := NewKeys([]Key{rkey}, 100)
	networkID := util.UUID().Bytes()
	amount := NewAmount(1000)

	ga, err := NewGenesisAccount(nodeKey, keys, amount, networkID)
	t.NoError(err)

	t.NoError(ga.IsValid(networkID))

	t.Implements((*base.Fact)(nil), ga.Fact())
	t.Implements((*operation.Operation)(nil), ga)
}

func TestGenesisAccount(t *testing.T) {
	suite.Run(t, new(testGenesisAccount))
}

func testGenesisAccountEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestOperationEncode)

	t.enc = enc
	t.newObject = func() interface{} {
		nodeKey := key.MustNewBTCPrivatekey()
		rpk := key.MustNewBTCPrivatekey()
		rkey, err := NewKey(rpk.Publickey(), 100)
		t.NoError(err)
		keys, _ := NewKeys([]Key{rkey}, 100)
		networkID := util.UUID().Bytes()
		amount := NewAmount(1000)

		ga, err := NewGenesisAccount(nodeKey, keys, amount, networkID)
		t.NoError(err)

		return ga
	}

	t.compare = func(a, b interface{}) {
		ca := a.(GenesisAccount)
		cb := b.(GenesisAccount)
		fact := ca.Fact().(GenesisAccountFact)
		ufact := cb.Fact().(GenesisAccountFact)

		t.True(fact.genesisNodeKey.Equal(ufact.genesisNodeKey))
		t.Equal(fact.amount, ufact.amount)

		t.True(fact.keys.Hash().Equal(ufact.keys.Hash()))
		for i := range fact.keys.Keys() {
			t.Equal(fact.keys.Keys()[i].Bytes(), ufact.keys.Keys()[i].Bytes())
		}
		t.Equal(fact.keys.Threshold(), ufact.keys.Threshold())
	}

	return t
}

func TestGenesisAccountEncodeJSON(t *testing.T) {
	suite.Run(t, testGenesisAccountEncode(jsonenc.NewEncoder()))
}

func TestGenesisAccountEncodeBSON(t *testing.T) {
	suite.Run(t, testGenesisAccountEncode(bsonenc.NewEncoder()))
}
