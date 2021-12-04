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

type testCurrencyDesign struct {
	suite.Suite
}

func (t *testCurrencyDesign) TestValid() {
	po := NewCurrencyPolicy(ZeroBig, NewNilFeeer())
	gc := NewCurrencyDesign(MustNewAmount(NewBig(33), CurrencyID("ABC")), NewTestAddress(), po)
	t.NoError(gc.IsValid(nil))
}

func (t *testCurrencyDesign) TestInValidAmount() {
	po := NewCurrencyPolicy(ZeroBig, NewNilFeeer())
	amount := NewAmount(NewBig(33), CurrencyID("abc"))

	gc := NewCurrencyDesign(amount, NewTestAddress(), po)
	t.Error(gc.IsValid(nil))
}

func (t *testCurrencyDesign) TestUnderZeroAmount() {
	po := NewCurrencyPolicy(ZeroBig, NewNilFeeer())
	amount := NewAmount(NewBig(0), CurrencyID("ABC"))
	t.NoError(amount.IsValid(nil))

	gc := NewCurrencyDesign(amount, NewTestAddress(), po)
	err := gc.IsValid(nil)
	t.Contains(err.Error(), "should be over zero")
}

func (t *testCurrencyDesign) TestAddAggregate() {
	de := NewCurrencyDesign(
		NewAmount(NewBig(33), CurrencyID("SHOWME")),
		NewTestAddress(),
		NewCurrencyPolicy(
			ZeroBig,
			NewFixedFeeer(MustAddress(util.UUID().String()), NewBig(44)),
		),
	)
	t.NoError(de.IsValid(nil))

	var err error
	de, err = de.AddAggregate(NewBig(10))
	t.NoError(err)

	de, err = de.AddAggregate(NewBig(1))
	t.NoError(err)

	_, err = de.AddAggregate(ZeroBig)
	t.Contains(err.Error(), "new aggregate not over zero")

	t.True(de.aggregate.Equal(NewBig(44)))
}

func TestCurrencyDesign(t *testing.T) {
	suite.Run(t, new(testCurrencyDesign))
}

func testCurrencyDesignEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestEncode)

	t.enc = enc
	t.newObject = func() interface{} {
		de := NewCurrencyDesign(
			NewAmount(NewBig(33), CurrencyID("SHOWME")),
			NewTestAddress(),
			NewCurrencyPolicy(
				ZeroBig,
				NewFixedFeeer(MustAddress(util.UUID().String()), NewBig(44)),
			),
		)
		de.BaseHinter = hint.NewBaseHinter(hint.NewHint(CurrencyDesignType, "v0.0.9"))
		t.NoError(de.IsValid(nil))

		return de
	}

	t.compare = func(a, b interface{}) {
		ca := a.(CurrencyDesign)
		cb := b.(CurrencyDesign)

		t.compareCurrencyDesign(ca, cb)
	}

	return t
}

func TestCurrencyDesignEncodeJSON(t *testing.T) {
	suite.Run(t, testCurrencyDesignEncode(jsonenc.NewEncoder()))
}

func TestCurrencyDesignEncodeBSON(t *testing.T) {
	suite.Run(t, testCurrencyDesignEncode(bsonenc.NewEncoder()))
}
