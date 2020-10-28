package currency

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testAddress struct {
	suite.Suite
}

func (t *testAddress) newKey(weight uint) Key {
	k, err := NewKey(key.MustNewBTCPrivatekey().Publickey(), weight)
	if err != nil {
		panic(err)
	}

	return k
}

func (t *testAddress) TestSingleKey() {
	k := t.newKey(100)
	keys, err := NewKeys([]Key{k}, 100)
	t.NoError(err)

	a, err := NewAddressFromKeys(keys)
	t.NoError(err)

	b, err := NewAddressFromKeys(keys)
	t.NoError(err)

	t.True(a.Equal(b))
}

func (t *testAddress) TestWrongKey() {
	keys := Keys{
		keys:      []Key{{k: key.MustNewBTCPrivatekey().Publickey(), w: 101}},
		threshold: 100,
		h:         valuehash.RandomSHA256(),
	}

	_, err := NewAddressFromKeys(keys)
	t.Contains(err.Error(), "invalid key")
}

func (t *testAddress) TestMultipleKey() {
	var ks []Key
	for i := 0; i < 3; i++ {
		ks = append(ks, t.newKey(30))
	}
	keys, err := NewKeys(ks, 90)
	t.NoError(err)

	a, err := NewAddressFromKeys(keys)
	t.NoError(err)

	b, err := NewAddressFromKeys(keys)
	t.NoError(err)

	t.Equal(a, b)
}

func (t *testAddress) TestMultipleKeyOrder() {
	var ks []Key
	for i := 0; i < 3; i++ {
		ks = append(ks, t.newKey(30))
	}

	keys, err := NewKeys(ks, 90)
	t.NoError(err)

	a, err := NewAddressFromKeys(keys)
	t.NoError(err)

	// set different order
	var nks []Key
	nks = append(nks, ks[2])
	nks = append(nks, ks[1])
	nks = append(nks, ks[0])

	newkeys, err := NewKeys(nks, 90)
	t.NoError(err)

	b, err := NewAddressFromKeys(newkeys)
	t.NoError(err)

	t.Equal(a, b)
}

func TestAddress(t *testing.T) {
	suite.Run(t, new(testAddress))
}
