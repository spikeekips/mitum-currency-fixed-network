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
	"github.com/spikeekips/mitum/util/hint"
	"github.com/stretchr/testify/suite"
)

type testCurrencyRegister struct {
	baseTest
}

func (t *testCurrencyRegister) TestNew() {
	token := util.UUID().Bytes()
	item := t.currencyDesign(NewBig(33), CurrencyID("SHOWME"))
	fact := NewCurrencyRegisterFact(token, item)

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

	op, err := NewCurrencyRegister(fact, fs, "")
	t.NoError(err)

	t.NoError(op.IsValid(nil))

	t.Implements((*base.Fact)(nil), op.Fact())
	t.Implements((*operation.Operation)(nil), op)

	t.Equal(fact, op.Fact())
}

func (t *testCurrencyRegister) TestZeroAmount() {
	token := util.UUID().Bytes()
	item := t.currencyDesign(NewBig(0), CurrencyID("SHOWME"))
	fact := NewCurrencyRegisterFact(token, item)

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

	op, err := NewCurrencyRegister(fact, fs, "")
	t.NoError(err)

	err = op.IsValid(nil)
	t.Contains(err.Error(), "currency balance should be over zero")
}

func (t *testCurrencyRegister) TestInvalidCurrencyID() {
	token := util.UUID().Bytes()
	item := t.currencyDesign(NewBig(33), CurrencyID("showme"))
	fact := NewCurrencyRegisterFact(token, item)

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

	op, err := NewCurrencyRegister(fact, fs, "")
	t.NoError(err)

	err = op.IsValid(nil)
	t.Contains(err.Error(), "wrong currency id")
}

func TestCurrencyRegister(t *testing.T) {
	suite.Run(t, new(testCurrencyRegister))
}

func testCurrencyRegisterEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestOperationEncode)

	t.enc = enc
	t.newObject = func() interface{} {
		token := util.UUID().Bytes()
		po := NewCurrencyPolicy(ZeroBig, NewNilFeeer())
		de := NewCurrencyDesign(NewAmount(NewBig(33), CurrencyID("SHOWME")), NewTestAddress(), po)
		fact := NewCurrencyRegisterFact(token, de)
		fact.BaseHinter = hint.NewBaseHinter(hint.NewHint(CurrencyRegisterFactType, "v0.0.9"))

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

		op, err := NewCurrencyRegister(fact, fs, "findme")
		t.NoError(err)
		op.BaseHinter = hint.NewBaseHinter(hint.NewHint(CurrencyRegisterType, "v0.0.9"))

		t.NoError(op.IsValid(nil))

		return op
	}

	t.compare = func(a, b interface{}) {
		ta := a.(CurrencyRegister)
		tb := b.(CurrencyRegister)

		t.True(ta.Hint().Equal(tb.Hint()))
		t.Equal(ta.Memo, tb.Memo)

		fact := ta.Fact().(CurrencyRegisterFact)
		ufact := tb.Fact().(CurrencyRegisterFact)

		t.True(fact.Hint().Equal(ufact.Hint()))

		ac := fact.currency
		bc := ufact.currency

		t.compareCurrencyDesign(ac, bc)
	}

	return t
}

func TestCurrencyRegisterEncodeJSON(t *testing.T) {
	suite.Run(t, testCurrencyRegisterEncode(jsonenc.NewEncoder()))
}

func TestCurrencyRegisterEncodeBSON(t *testing.T) {
	suite.Run(t, testCurrencyRegisterEncode(bsonenc.NewEncoder()))
}
