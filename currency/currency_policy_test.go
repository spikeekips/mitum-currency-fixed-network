package currency

import (
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/stretchr/testify/suite"
)

type testCurrencyPolicy struct {
	suite.Suite
}

func (t *testCurrencyPolicy) TestValid() {
	po := NewCurrencyPolicy(ZeroBig, NewNilFeeer())
	t.NoError(po.IsValid(nil))
}

func (t *testCurrencyPolicy) TestInValidNewAccountMinBalance() {
	po := NewCurrencyPolicy(NilBig, NewNilFeeer())
	err := po.IsValid(nil)
	t.Contains(err.Error(), "NewAccountMinBalance under zero")
}

func TestCurrencyPolicy(t *testing.T) {
	suite.Run(t, new(testCurrencyPolicy))
}

func testCurrencyPolicyEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestEncode)

	t.enc = enc
	t.newObject = func() interface{} {
		po := NewCurrencyPolicy(ZeroBig, NewFixedFeeer(MustAddress(util.UUID().String()), NewBig(33)))
		po.BaseHinter = hint.NewBaseHinter(hint.NewHint(CurrencyPolicyType, "v0.0.9"))

		return po
	}

	t.compare = func(a, b interface{}) {
		ca := a.(CurrencyPolicy)
		cb := b.(CurrencyPolicy)

		t.True(ca.Hint().Equal(cb.Hint()))
		t.Equal(ca, cb)
	}

	return t
}

func TestCurrencyPolicyEncodeJSON(t *testing.T) {
	suite.Run(t, testCurrencyPolicyEncode(jsonenc.NewEncoder()))
}

func TestCurrencyPolicyEncodeBSON(t *testing.T) {
	suite.Run(t, testCurrencyPolicyEncode(bsonenc.NewEncoder()))
}
