package currency

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/stretchr/testify/suite"
)

type testCurrencyPolicyUpdater struct {
	baseTest
}

func (t *testCurrencyPolicyUpdater) TestNew() {
	token := util.UUID().Bytes()
	po := NewCurrencyPolicy(ZeroBig, NewFixedFeeer(MustAddress(util.UUID().String()), NewBig(44)))

	fact := NewCurrencyPolicyUpdaterFact(token, t.cid, po)

	var fs []base.FactSign

	for _, pk := range []key.Privatekey{
		key.MustNewBTCPrivatekey(),
		key.MustNewBTCPrivatekey(),
		key.MustNewBTCPrivatekey(),
	} {
		sig, err := base.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, base.NewBaseFactSign(pk.Publickey(), sig))
	}

	op, err := NewCurrencyPolicyUpdater(fact, fs, "")
	t.NoError(err)

	t.NoError(op.IsValid(nil))

	t.Implements((*base.Fact)(nil), op.Fact())
	t.Implements((*operation.Operation)(nil), op)

	t.Equal(fact, op.Fact())
}

func (t *testCurrencyPolicyUpdater) TestWithInvalidPolicy() {
	token := util.UUID().Bytes()
	po := NewCurrencyPolicy(ZeroBig, NewFixedFeeer(MustAddress(util.UUID().String()), NewBig(-1)))

	err := po.IsValid(nil)
	t.Contains(err.Error(), "fixed feeer amount under zero")

	fact := NewCurrencyPolicyUpdaterFact(token, t.cid, po)

	var fs []base.FactSign

	for _, pk := range []key.Privatekey{
		key.MustNewBTCPrivatekey(),
		key.MustNewBTCPrivatekey(),
		key.MustNewBTCPrivatekey(),
	} {
		sig, err := base.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, base.NewBaseFactSign(pk.Publickey(), sig))
	}

	op, err := NewCurrencyPolicyUpdater(fact, fs, "")
	t.NoError(err)

	err = op.IsValid(nil)
	t.Contains(err.Error(), "fixed feeer amount under zero")
}

func TestCurrencyPolicyUpdater(t *testing.T) {
	suite.Run(t, new(testCurrencyPolicyUpdater))
}

func testCurrencyPolicyUpdaterEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestOperationEncode)

	t.enc = enc
	t.newObject = func() interface{} {
		token := util.UUID().Bytes()
		po := NewCurrencyPolicy(NewBig(44), NewFixedFeeer(MustAddress(util.UUID().String()), NewBig(44)))
		t.NoError(po.IsValid(nil))

		fact := NewCurrencyPolicyUpdaterFact(token, CurrencyID("FINDME"), po)

		var fs []base.FactSign

		for _, pk := range []key.Privatekey{
			key.MustNewBTCPrivatekey(),
			key.MustNewBTCPrivatekey(),
			key.MustNewBTCPrivatekey(),
		} {
			sig, err := base.NewFactSignature(pk, fact, nil)
			t.NoError(err)

			fs = append(fs, base.NewBaseFactSign(pk.Publickey(), sig))
		}

		op, err := NewCurrencyPolicyUpdater(fact, fs, "")
		t.NoError(err)

		t.NoError(op.IsValid(nil))

		return op
	}

	t.compare = func(a, b interface{}) {
		ta := a.(CurrencyPolicyUpdater)
		tb := b.(CurrencyPolicyUpdater)

		t.Equal(ta.Memo, tb.Memo)

		fact := ta.Fact().(CurrencyPolicyUpdaterFact)
		ufact := tb.Fact().(CurrencyPolicyUpdaterFact)

		t.Equal(fact.cid, ufact.cid)
		t.Equal(fact.policy, ufact.policy)
	}

	return t
}

func TestCurrencyPolicyUpdaterEncodeJSON(t *testing.T) {
	suite.Run(t, testCurrencyPolicyUpdaterEncode(jsonenc.NewEncoder()))
}

func TestCurrencyPolicyUpdaterEncodeBSON(t *testing.T) {
	suite.Run(t, testCurrencyPolicyUpdaterEncode(bsonenc.NewEncoder()))
}
