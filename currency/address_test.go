package currency

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testAddress struct {
	suite.Suite
}

func (t *testAddress) newKey(weight uint) BaseAccountKey {
	k, err := NewBaseAccountKey(key.NewBasePrivatekey().Publickey(), weight)
	if err != nil {
		panic(err)
	}

	return k
}

func (t *testAddress) TestSingleKey() {
	k := t.newKey(100)
	keys, err := NewBaseAccountKeys([]AccountKey{k}, 100)
	t.NoError(err)

	a, err := NewAddressFromKeys(keys)
	t.NoError(err)

	b, err := NewAddressFromKeys(keys)
	t.NoError(err)

	t.True(a.Equal(b))
}

func (t *testAddress) TestWrongKey() {
	keys := BaseAccountKeys{
		keys:      []AccountKey{BaseAccountKey{k: key.NewBasePrivatekey().Publickey(), w: 101}},
		threshold: 100,
		h:         valuehash.RandomSHA256(),
	}

	_, err := NewAddressFromKeys(keys)
	t.Contains(err.Error(), "invalid key")
}

func (t *testAddress) TestMultipleKey() {
	var ks []AccountKey
	for i := 0; i < 3; i++ {
		ks = append(ks, t.newKey(30))
	}
	keys, err := NewBaseAccountKeys(ks, 90)
	t.NoError(err)

	a, err := NewAddressFromKeys(keys)
	t.NoError(err)

	b, err := NewAddressFromKeys(keys)
	t.NoError(err)

	t.Equal(a, b)
}

func (t *testAddress) TestMultipleKeyOrder() {
	var ks []AccountKey
	for i := 0; i < 3; i++ {
		ks = append(ks, t.newKey(30))
	}

	keys, err := NewBaseAccountKeys(ks, 90)
	t.NoError(err)

	a, err := NewAddressFromKeys(keys)
	t.NoError(err)

	// set different order
	var nks []AccountKey
	nks = append(nks, ks[2])
	nks = append(nks, ks[1])
	nks = append(nks, ks[0])

	newkeys, err := NewBaseAccountKeys(nks, 90)
	t.NoError(err)

	b, err := NewAddressFromKeys(newkeys)
	t.NoError(err)

	t.Equal(a, b)
}

func isZeroAddress(cid CurrencyID, address base.Address) bool {
	return cid.String()+ZeroAddressSuffix+AddressType.String() == address.String()
}

func (t *testAddress) TestZeroAddress() {
	cid := CurrencyID("XYZ")
	ad := ZeroAddress(cid)
	t.Equal("XYZ-X"+AddressType.String(), ad.String())

	t.True(isZeroAddress(cid, ad))
}

func TestAddress(t *testing.T) {
	suite.Run(t, new(testAddress))
}
