package currency

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/stretchr/testify/suite"
)

type testSuffrageInflationFact struct {
	baseTest
}

func (t *testSuffrageInflationFact) TestNew() {
	token := util.UUID().Bytes()
	item := NewSuffrageInflationItem(base.RandomStringAddress(), NewAmount(NewBig(33), CurrencyID("SHOWME")))
	fact := NewSuffrageInflationFact(token, []SuffrageInflationItem{item})

	t.NoError(fact.IsValid(nil))

	t.Implements((*base.Fact)(nil), fact)
}

func (t *testSuffrageInflationFact) TestTooManyItems() {
	token := util.UUID().Bytes()

	items := make([]SuffrageInflationItem, maxSuffrageInflationItem+1)
	for i := int64(0); i < int64(maxSuffrageInflationItem+1); i++ {
		item := NewSuffrageInflationItem(base.RandomStringAddress(), NewAmount(NewBig(i+1), CurrencyID("SHOWME")))

		items[i] = item
	}
	fact := NewSuffrageInflationFact(token, items)

	err := fact.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "too many items")
}

func (t *testSuffrageInflationFact) TestWrongReceiver() {
	token := util.UUID().Bytes()
	item := NewSuffrageInflationItem(nil, NewAmount(NewBig(33), CurrencyID("SHOWME")))
	fact := NewSuffrageInflationFact(token, []SuffrageInflationItem{item})

	err := fact.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
}

func (t *testSuffrageInflationFact) TestZeroAmount() {
	token := util.UUID().Bytes()
	item := NewSuffrageInflationItem(base.RandomStringAddress(), NewAmount(NewBig(0), CurrencyID("SHOWME")))
	fact := NewSuffrageInflationFact(token, []SuffrageInflationItem{item})

	err := fact.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
}

func TestSuffrageInflationFact(t *testing.T) {
	suite.Run(t, new(testSuffrageInflationFact))
}

type testSuffrageInflation struct {
	baseTest
}

func (t *testSuffrageInflation) item(big Big, cid CurrencyID, receiver base.Address) SuffrageInflationItem {
	return NewSuffrageInflationItem(receiver, NewAmount(big, cid))
}

func (t *testSuffrageInflation) TestNew() {
	token := util.UUID().Bytes()
	item := t.item(NewBig(33), CurrencyID("SHOWME"), base.RandomStringAddress())
	fact := NewSuffrageInflationFact(token, []SuffrageInflationItem{item})

	var fs []base.FactSign

	for _, pk := range []key.Privatekey{
		key.NewBasePrivatekey(),
		key.NewBasePrivatekey(),
		key.NewBasePrivatekey(),
	} {
		sig, err := base.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, base.NewBaseFactSign(pk.Publickey(), sig))
	}

	op, err := NewSuffrageInflation(fact, fs, "")
	t.NoError(err)

	t.NoError(op.IsValid(nil))

	t.Implements((*base.Fact)(nil), op.Fact())
	t.Implements((*operation.Operation)(nil), op)

	t.Equal(fact, op.Fact())
}

func (t *testSuffrageInflation) TestDuplicatedItem() {
	token := util.UUID().Bytes()

	var items []SuffrageInflationItem
	for i := 0; i < 3; i++ {
		item := t.item(NewBig(33), CurrencyID("SHOWME"), base.RandomStringAddress())
		items = append(items, item)
	}

	item := t.item(NewBig(44), items[2].amount.Currency(), items[2].receiver)
	items = append(items, item)

	fact := NewSuffrageInflationFact(token, items)

	var fs []base.FactSign

	for _, pk := range []key.Privatekey{
		key.NewBasePrivatekey(),
		key.NewBasePrivatekey(),
		key.NewBasePrivatekey(),
	} {
		sig, err := base.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, base.NewBaseFactSign(pk.Publickey(), sig))
	}

	op, err := NewSuffrageInflation(fact, fs, "")
	t.NoError(err)

	err = op.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "duplicated item found in SuffrageInflationFact")
}

func TestSuffrageInflation(t *testing.T) {
	suite.Run(t, new(testSuffrageInflation))
}

func testSuffrageInflationEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestOperationEncode)
	t.enc = enc
	t.newObject = func() interface{} {
		token := util.UUID().Bytes()

		items := []SuffrageInflationItem{
			NewSuffrageInflationItem(base.RandomStringAddress(), NewAmount(NewBig(33), CurrencyID("SHOWME"))),
			NewSuffrageInflationItem(base.RandomStringAddress(), NewAmount(NewBig(44), CurrencyID("FINDME"))),
		}

		fact := NewSuffrageInflationFact(token, items)

		var fs []base.FactSign

		for _, pk := range []key.Privatekey{
			key.NewBasePrivatekey(),
			key.NewBasePrivatekey(),
			key.NewBasePrivatekey(),
		} {
			sig, err := base.NewFactSignature(pk, fact, nil)
			t.NoError(err)

			fs = append(fs, base.NewBaseFactSign(pk.Publickey(), sig))
		}

		op, err := NewSuffrageInflation(fact, fs, "findme")
		t.NoError(err)

		t.NoError(op.IsValid(nil))

		return op
	}

	t.compare = func(a, b interface{}) {
		ta := a.(SuffrageInflation)
		tb := b.(SuffrageInflation)

		t.Equal(ta.Memo, tb.Memo)

		fact := ta.Fact().(SuffrageInflationFact)
		ufact := tb.Fact().(SuffrageInflationFact)

		t.True(fact.h.Equal(ufact.h))
		t.Equal(fact.token, ufact.token)

		for i := range fact.items {
			ai := fact.items[i]
			bi := ufact.items[i]
			t.True(ai.receiver.Equal(bi.receiver))
			t.True(ai.amount.Equal(bi.amount))
		}
	}

	return t
}

func TestSuffrageInflationEncodeJSON(t *testing.T) {
	suite.Run(t, testSuffrageInflationEncode(jsonenc.NewEncoder()))
}

func TestSuffrageInflationEncodeBSON(t *testing.T) {
	suite.Run(t, testSuffrageInflationEncode(bsonenc.NewEncoder()))
}
