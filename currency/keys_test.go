package currency

import (
	"fmt"
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
	k, err := NewKey(key.MustNewBTCPrivatekey().Publickey(), 50)
	t.NoError(err)
	t.NoError(k.IsValid(nil))
}

func (t *testKey) TestZeroWeight() {
	_, err := NewKey(key.MustNewBTCPrivatekey().Publickey(), 0)
	t.Contains(err.Error(), "invalid key weight")
}

func (t *testKey) Test100Weight() {
	k, err := NewKey(key.MustNewBTCPrivatekey().Publickey(), 100)
	t.NoError(err)
	t.NoError(k.IsValid(nil))
}

func (t *testKey) TestOver100Weight() {
	_, err := NewKey(key.MustNewBTCPrivatekey().Publickey(), 101)
	t.Contains(err.Error(), "invalid key weight")
}

func TestKey(t *testing.T) {
	suite.Run(t, new(testKey))
}

type testKeys struct {
	suite.Suite
}

func (t *testKeys) newKey(pub key.Publickey, w uint) Key {
	k, err := NewKey(pub, w)
	if err != nil {
		panic(err)
	}

	return k
}

func (t *testKeys) TestNew() {
	keys := []Key{
		t.newKey(key.MustNewBTCPrivatekey().Publickey(), 50),
		t.newKey(key.MustNewBTCPrivatekey().Publickey(), 50),
	}

	ks, err := NewKeys(keys, 100)
	t.NoError(err)
	t.NotNil(ks.Hash())
	t.NoError(ks.IsValid(nil))
	t.Equal(2, len(ks.Keys()))
}

func (t *testKeys) TestSorting() {
	k0 := t.newKey(key.MustNewBTCPrivatekey().Publickey(), 50)
	k1 := t.newKey(key.MustNewBTCPrivatekey().Publickey(), 50)

	ks0, _ := NewKeys([]Key{k0, k1}, 100)
	ks1, _ := NewKeys([]Key{k1, k0}, 100)

	t.True(ks0.Hash().Equal(ks1.Hash()))
}

func (t *testKeys) TestWeightOver100() {
	keys := []Key{
		t.newKey(key.MustNewBTCPrivatekey().Publickey(), 50),
		t.newKey(key.MustNewBTCPrivatekey().Publickey(), 30),
		t.newKey(key.MustNewBTCPrivatekey().Publickey(), 50),
	}

	_, err := NewKeys(keys, 100)
	t.NoError(err)
}

func (t *testKeys) TestKeysOver100() {
	keys := make([]Key, MaxKeyInKeys+1)
	for i := 0; i < MaxKeyInKeys+1; i++ {
		keys[i] = t.newKey(key.MustNewBTCPrivatekey().Publickey(), 50)
	}

	_, err := NewKeys(keys, 100)
	t.Contains(err.Error(), fmt.Sprintf("keys over %d", MaxKeyInKeys))
}

func (t *testKeys) TestBadThreshold() {
	keys := []Key{t.newKey(key.MustNewBTCPrivatekey().Publickey(), 50)}

	_, err := NewKeys(keys, 101)
	t.Contains(err.Error(), "invalid threshold")

	_, err = NewKeys(keys, 0)
	t.Contains(err.Error(), "invalid threshold")
}

func (t *testKeys) TestCheckSigning() {
	pk := key.MustNewBTCPrivatekey()

	keys := []Key{
		t.newKey(pk.Publickey(), 23),
		t.newKey(key.MustNewBTCPrivatekey().Publickey(), 77),
	}

	ks, err := NewKeys(keys, 100)
	t.NoError(err)
	t.NoError(ks.IsValid(nil))
}

func (t *testKeys) TestSingleKeyUnderThreshold() {
	pk := key.MustNewBTCPrivatekey()

	keys := []Key{t.newKey(pk.Publickey(), 23)}

	_, err := NewKeys(keys, 100)
	t.Contains(err.Error(), "sum of weight under threshold")
}

func (t *testKeys) TestEqual() {
	keys0 := []Key{
		t.newKey(key.MustNewBTCPrivatekey().Publickey(), 50),
		t.newKey(key.MustNewBTCPrivatekey().Publickey(), 50),
	}

	ks0, err := NewKeys(keys0, 100)
	t.NoError(err)
	t.NotNil(ks0.Hash())
	t.NoError(ks0.IsValid(nil))

	ks1, err := NewKeys(keys0, 100)
	t.NoError(err)
	t.NotNil(ks1.Hash())
	t.NoError(ks1.IsValid(nil))

	t.True(ks0.Equal(ks1))

	// different []Key
	keys2 := []Key{
		t.newKey(key.MustNewBTCPrivatekey().Publickey(), 50),
		t.newKey(key.MustNewBTCPrivatekey().Publickey(), 50),
	}
	ks2, err := NewKeys(keys2, ks0.Threshold())
	t.NoError(err)
	t.NotNil(ks2.Hash())
	t.NoError(ks2.IsValid(nil))

	t.False(ks0.Equal(ks2))

	// different threshold
	ks3, err := NewKeys(keys0, 99)
	t.NoError(err)
	t.NotNil(ks3.Hash())
	t.NoError(ks3.IsValid(nil))

	t.False(ks0.Equal(ks3))
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

	encs.AddHinter(key.BTCPublickeyHinter)
	encs.AddHinter(Key{})
	encs.AddHinter(Keys{})
}

func (t *testKeysEncode) TestMarshal() {
	ak, err := NewKey(key.MustNewBTCPrivatekey().Publickey(), 50)
	t.NoError(err)
	bk, err := NewKey(key.MustNewBTCPrivatekey().Publickey(), 50)
	t.NoError(err)
	keys := []Key{ak, bk}

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
