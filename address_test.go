package mc

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
	a, err := NewAddressFromKeys([]Key{k})
	t.NoError(err)

	t.Equal(k.Key().String(), a.String())
}

func (t *testAddress) TestWrongKey() {
	k := t.newKey(101)
	_, err := NewAddressFromKeys([]Key{k})
	t.Contains(err.Error(), "invalid key")
}

func (t *testAddress) TestMultipleKey() {
	var ks []Key
	for i := 0; i < 3; i++ {
		ks = append(ks, t.newKey(30))
	}
	a, err := NewAddressFromKeys(ks)
	t.NoError(err)

	// set different order
	var nks []Key
	nks = append(nks, ks[2])
	nks = append(nks, ks[1])
	nks = append(nks, ks[0])

	b, err := NewAddressFromKeys(ks)
	t.NoError(err)

	t.Equal(a, b)
}

func TestAddress(t *testing.T) {
	suite.Run(t, new(testAddress))
}
