package currency

import (
	"testing"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/stretchr/testify/suite"
)

type testAccount struct {
	suite.Suite
}

func (t *testAccount) TestNew() {
	priv := key.MustNewBTCPrivatekey()
	key, err := NewKey(priv.Publickey(), 100)
	t.NoError(err)
	keys, err := NewKeys([]Key{key}, 100)
	t.NoError(err)

	address, err := NewAddress(util.UUID().String())
	t.NoError(err)

	ac, err := NewAccount(address, keys)
	t.NoError(err)

	t.True(ac.Address().Equal(address))
	t.True(ac.Keys().Equal(keys))
}

func (t *testAccount) TestNewFromKeys() {
	priv := key.MustNewBTCPrivatekey()
	key, err := NewKey(priv.Publickey(), 100)
	t.NoError(err)
	keys, err := NewKeys([]Key{key}, 100)
	t.NoError(err)

	ac, err := NewAccountFromKeys(keys)
	t.NoError(err)

	af, err := NewAddressFromKeys(keys)
	t.NoError(err)

	t.True(ac.Address().Equal(af))
	t.True(ac.Keys().Equal(keys))
}

func TestAccount(t *testing.T) {
	suite.Run(t, new(testAccount))
}

func testAccountEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestEncode)

	t.enc = enc
	t.newObject = func() interface{} {
		priv := key.MustNewBTCPrivatekey()
		key, err := NewKey(priv.Publickey(), 100)
		t.NoError(err)
		keys, err := NewKeys([]Key{key}, 100)
		t.NoError(err)

		ac, err := NewAccountFromKeys(keys)
		t.NoError(err)

		return ac
	}

	t.compare = func(a, b interface{}) {
		ca := a.(Account)
		cb := b.(Account)

		t.True(ca.Address().Equal(cb.Address()))
		t.True(ca.Keys().Equal(cb.Keys()))
	}

	return t
}

func TestAccountEncodeJSON(t *testing.T) {
	suite.Run(t, testAccountEncode(jsonenc.NewEncoder()))
}

func TestAccountEncodeBSON(t *testing.T) {
	suite.Run(t, testAccountEncode(bsonenc.NewEncoder()))
}
