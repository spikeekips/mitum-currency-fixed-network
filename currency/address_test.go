package currency

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base/key"
)

type testAddress struct {
	suite.Suite
}

func (t *testAddress) newKey(weight uint) Key {
	return NewKey(key.MustNewBTCPrivatekey().Publickey(), weight)
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
	k := t.newKey(101)
	keys, err := NewKeys([]Key{k}, 100)
	t.NoError(err)

	_, err = NewAddressFromKeys(keys)
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

	b, err := NewAddressFromKeys(keys)
	t.NoError(err)

	t.Equal(a, b)
}

func TestAddress(t *testing.T) {
	suite.Run(t, new(testAddress))
}
