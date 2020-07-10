package mc

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/stretchr/testify/suite"
)

type testGenesisAccount struct {
	suite.Suite
}

func (t *testGenesisAccount) TestNew() {
	nodeKey := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()
	rkey := NewKey(rpk.Publickey(), 100)
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

type testGenesisAccountEncode struct {
	suite.Suite

	enc encoder.Encoder
}

func (t *testGenesisAccountEncode) SetupSuite() {
	encs := encoder.NewEncoders()
	encs.AddEncoder(t.enc)

	encs.AddHinter(key.BTCPublickey{})
	encs.AddHinter(Address(""))
	encs.AddHinter(operation.BaseFactSign{})

	encs.AddHinter(Key{})
	encs.AddHinter(Keys{})
	encs.AddHinter(GenesisAccountFact{})
	encs.AddHinter(GenesisAccount{})
}

func (t *testGenesisAccountEncode) TestEncode() {
	nodeKey := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()
	rkey := NewKey(rpk.Publickey(), 100)
	keys, _ := NewKeys([]Key{rkey}, 100)
	networkID := util.UUID().Bytes()
	amount := NewAmount(1000)

	ga, err := NewGenesisAccount(nodeKey, keys, amount, networkID)
	t.NoError(err)
	t.NoError(ga.IsValid(networkID))

	b, err := t.enc.Marshal(ga)
	t.NoError(err)

	hinter, err := t.enc.DecodeByHint(b)
	t.NoError(err)

	uga, ok := hinter.(GenesisAccount)
	t.True(ok)
	t.NoError(uga.IsValid(networkID))

	fact := ga.Fact().(GenesisAccountFact)
	ufact := uga.Fact().(GenesisAccountFact)
	t.True(fact.h.Equal(ufact.h))
	t.Equal(fact.token, ufact.token)
	t.True(fact.genesisNodeKey.Equal(ufact.genesisNodeKey))
	t.Equal(fact.amount, ufact.amount)

	t.True(ga.Hash().Equal(uga.Hash()))

	t.True(fact.keys.Hash().Equal(ufact.keys.Hash()))
	t.Equal(fact.keys.Keys(), ufact.keys.Keys())
	t.Equal(fact.keys.Threshold(), ufact.keys.Threshold())

	for i := range ga.Signs() {
		a := ga.Signs()[i]
		b := uga.Signs()[i]

		t.True(a.Signer().Equal(b.Signer()))
		t.Equal(a.Signature(), b.Signature())
		t.Equal(localtime.RFC3339(a.SignedAt()), localtime.RFC3339(b.SignedAt()))
	}
}

func TestGenesisAccountEncodeJSON(t *testing.T) {
	b := new(testGenesisAccountEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestGenesisAccountEncodeBSON(t *testing.T) {
	b := new(testGenesisAccountEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
