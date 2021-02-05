package currency

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type testFeeOperation struct {
	baseTest
}

func (t *testFeeOperation) TestNew() {
	cid := CurrencyID("SHOWME")
	fee := NewBig(33)

	height := base.Height(3)
	fact := NewFeeOperationFact(height, map[CurrencyID]Big{cid: fee})

	op := NewFeeOperation(fact)

	t.NoError(op.IsValid(nil))

	t.Implements((*base.Fact)(nil), op.Fact())
	t.Implements((*operation.Operation)(nil), op)

	nfact := op.Fact().(FeeOperationFact)
	t.Equal(fee, nfact.Amounts()[0].Big())
	t.Equal(cid, nfact.Amounts()[0].Currency())
}

func TestFeeOperation(t *testing.T) {
	suite.Run(t, new(testFeeOperation))
}

func testFeeOperationEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestOperationEncode)

	t.enc = enc
	t.newObject = func() interface{} {
		fact := NewFeeOperationFact(base.Height(3), map[CurrencyID]Big{CurrencyID("SHOWME"): NewBig(33)})

		return NewFeeOperation(fact)
	}

	t.compare = func(a, b interface{}) {
		ca := a.(FeeOperation)
		cb := b.(FeeOperation)
		fact := ca.Fact().(FeeOperationFact)
		ufact := cb.Fact().(FeeOperationFact)

		t.Equal(fact.token, ufact.token)

		t.Equal(len(fact.Amounts()), len(ufact.Amounts()))

		for i := range fact.Amounts() {
			am := fact.Amounts()[i]
			bm := ufact.Amounts()[i]

			t.True(am.Equal(bm))
		}
	}

	return t
}

func TestFeeOperationEncodeJSON(t *testing.T) {
	suite.Run(t, testFeeOperationEncode(jsonenc.NewEncoder()))
}

func TestFeeOperationEncodeBSON(t *testing.T) {
	suite.Run(t, testFeeOperationEncode(bsonenc.NewEncoder()))
}
