package currency

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type testFeeOperation struct {
	baseTest
}

func (t *testFeeOperation) TestNew() {
	rpk := key.MustNewBTCPrivatekey()
	rkey, err := NewKey(rpk.Publickey(), 100)
	t.NoError(err)
	rkeys, err := NewKeys([]Key{rkey}, 100)
	t.NoError(err)
	receiver, err := NewAddressFromKeys(rkeys)
	t.NoError(err)

	fa := NewFixedFeeAmount(NewAmount(7))

	height := base.Height(3)
	fee := NewAmount(33)
	fact := NewFeeOperationFact(fa, height, receiver, fee)
	t.Equal(fa.Verbose(), fact.fa)

	op := NewFeeOperation(fact)

	t.NoError(op.IsValid(nil))

	t.Implements((*base.Fact)(nil), op.Fact())
	t.Implements((*operation.Operation)(nil), op)

	nfact := op.Fact().(FeeOperationFact)
	t.True(receiver.Equal(nfact.Receiver()))
	t.Equal(fee, nfact.Fee())
}

func TestFeeOperation(t *testing.T) {
	suite.Run(t, new(testFeeOperation))
}

func testFeeOperationEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestOperationEncode)

	t.enc = enc
	t.newObject = func() interface{} {
		rkey, err := NewKey(key.MustNewBTCPrivatekey().Publickey(), 100)
		t.NoError(err)
		rkeys, err := NewKeys([]Key{rkey}, 100)
		t.NoError(err)
		receiver, err := NewAddressFromKeys(rkeys)
		t.NoError(err)

		fact := NewFeeOperationFact(NewFixedFeeAmount(NewAmount(7)), base.Height(3), receiver, NewAmount(33))

		return NewFeeOperation(fact)
	}

	t.compare = func(a, b interface{}) {
		ca := a.(FeeOperation)
		cb := b.(FeeOperation)
		fact := ca.Fact().(FeeOperationFact)
		ufact := cb.Fact().(FeeOperationFact)

		t.True(fact.receiver.Equal(ufact.receiver))
		t.Equal(fact.token, ufact.token)
		t.Equal(fact.fa, ufact.fa)
		t.True(fact.fee.Equal(ufact.fee))
	}

	return t
}

func TestFeeOperationEncodeJSON(t *testing.T) {
	suite.Run(t, testFeeOperationEncode(jsonenc.NewEncoder()))
}

func TestFeeOperationEncodeBSON(t *testing.T) {
	suite.Run(t, testFeeOperationEncode(bsonenc.NewEncoder()))
}
