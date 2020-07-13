package mc

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type testKey struct {
	suite.Suite
}

func (t *testKey) TestNew() {
	k := NewKey(key.MustNewBTCPrivatekey().Publickey(), 50)
	t.NoError(k.IsValid(nil))
}

func (t *testKey) TestZeroWeight() {
	k := NewKey(key.MustNewBTCPrivatekey().Publickey(), 0)
	err := k.IsValid(nil)
	t.Contains(err.Error(), "invalid key weight")
}

func (t *testKey) Test100Weight() {
	k := NewKey(key.MustNewBTCPrivatekey().Publickey(), 100)
	t.NoError(k.IsValid(nil))
}

func (t *testKey) TestOver100Weight() {
	k := NewKey(key.MustNewBTCPrivatekey().Publickey(), 101)
	err := k.IsValid(nil)
	t.Contains(err.Error(), "invalid key weight")
}

func TestKey(t *testing.T) {
	suite.Run(t, new(testKey))
}

type testKeys struct {
	suite.Suite
}

func (t *testKeys) TestNew() {
	keys := []Key{
		NewKey(key.MustNewBTCPrivatekey().Publickey(), 50),
		NewKey(key.MustNewBTCPrivatekey().Publickey(), 50),
	}

	ks, err := NewKeys(keys, 100)
	t.NoError(err)
	t.NotNil(ks.Hash())
	t.NoError(ks.IsValid(nil))
	t.Equal(2, len(ks.Keys()))
}

func (t *testKeys) TestWeightOver100() {
	keys := []Key{
		NewKey(key.MustNewBTCPrivatekey().Publickey(), 50),
		NewKey(key.MustNewBTCPrivatekey().Publickey(), 30),
		NewKey(key.MustNewBTCPrivatekey().Publickey(), 50),
	}

	ks, err := NewKeys(keys, 100)
	t.NoError(err)

	err = ks.IsValid(nil)
	t.Contains(err.Error(), "over 100")
}

func (t *testKeys) TestKeysOver100() {
	keys := make([]Key, 101)
	for i := 0; i < 101; i++ {
		keys[i] = NewKey(key.MustNewBTCPrivatekey().Publickey(), 50)
	}

	ks, err := NewKeys(keys, 100)
	t.NoError(err)

	err = ks.IsValid(nil)
	t.Contains(err.Error(), "keys over 100")
}

func (t *testKeys) TestBadThreshold() {
	keys := []Key{NewKey(key.MustNewBTCPrivatekey().Publickey(), 50)}

	ks, err := NewKeys(keys, 101)
	t.NoError(err)
	err = ks.IsValid(nil)
	t.Contains(err.Error(), "invalid threshold")

	ks, err = NewKeys(keys, 0)
	t.NoError(err)
	err = ks.IsValid(nil)
	t.Contains(err.Error(), "invalid threshold")
}

func (t *testKeys) TestCheckSigning() {
	pk := key.MustNewBTCPrivatekey()

	keys := []Key{
		NewKey(pk.Publickey(), 23),
		NewKey(key.MustNewBTCPrivatekey().Publickey(), 77),
	}

	ks, err := NewKeys(keys, 100)
	t.NoError(err)
	t.NoError(ks.IsValid(nil))
}

func (t *testKeys) TestSingleKeyUnderThreshold() {
	pk := key.MustNewBTCPrivatekey()

	keys := []Key{NewKey(pk.Publickey(), 23)}

	ks, err := NewKeys(keys, 100)
	t.NoError(err)

	err = ks.IsValid(nil)
	t.Contains(err.Error(), "sum of weight under threshold")
}

func TestKeys(t *testing.T) {
	suite.Run(t, new(testKeys))
}

type testKeysEncode struct {
	suite.Suite
	enc encoder.Encoder
}

func (t *testKeysEncode) SetupSuite() {
	encs := encoder.NewEncoders()
	encs.AddEncoder(t.enc)

	encs.AddHinter(key.BTCPublickey{})
	encs.AddHinter(Key{})
	encs.AddHinter(Keys{})
}

func (t *testKeysEncode) TestMarshal() {
	keys := []Key{
		NewKey(key.MustNewBTCPrivatekey().Publickey(), 50),
		NewKey(key.MustNewBTCPrivatekey().Publickey(), 50),
	}

	ks, err := NewKeys(keys, 100)
	t.NoError(err)

	b, err := t.enc.Marshal(ks)
	t.NoError(err)

	hinter, err := t.enc.DecodeByHint(b)
	t.NoError(err)
	uks, ok := hinter.(Keys)
	t.True(ok)

	t.True(ks.Hash().Equal(uks.Hash()))

	for i := range ks.Keys() {
		a := ks.Keys()[i]
		b := uks.Keys()[i]

		t.Equal(a.Weight(), b.Weight())
		t.True(a.Key().Equal(b.Key()))
	}
}

func TestKeysEncodeJSON(t *testing.T) {
	b := new(testKeysEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestKeysEncodeBSON(t *testing.T) {
	b := new(testKeysEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
