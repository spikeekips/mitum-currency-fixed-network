package currency

import (
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/stretchr/testify/suite"
)

type testFeeer struct {
	suite.Suite
}

func (t *testFeeer) TestFixedFeer() {
	cases := []struct {
		name   string
		fee    string
		big    string
		result string
		err    string
	}{
		{
			name:   "10 > 10",
			fee:    "10",
			big:    "10",
			result: "10",
		},
		{
			name:   "10 > 5",
			fee:    "5",
			big:    "10",
			result: "5",
		},
		{
			name:   "5 > 10",
			fee:    "10",
			big:    "5",
			result: "10",
		},
		{
			name:   "5 > 0",
			fee:    "0",
			big:    "5",
			result: "0",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				fee, err := NewBigFromString(c.fee)
				t.NoError(err)

				big, err := NewBigFromString(c.big)
				t.NoError(err)

				receiver := MustAddress(util.UUID().String())
				fa := NewFixedFeeer(receiver, fee)
				t.NoError(fa.IsValid(nil))

				t.Equal(fee, fa.Min())

				result, err := fa.Fee(big)
				if len(c.err) > 0 {
					t.Contains(err.Error(), c.err, "%d: %v; %v != %v", i, c.name, c.err, err)
				} else if err != nil {
					t.NoError(err, "%d: %v; %v != %v", i, c.name, c.err, err)
				} else {
					t.Equal(c.result, result.String(), "%d: %v; %v != %v", i, c.name, c.result, result.String())
				}
			},
		)
	}
}

func (t *testFeeer) TestRatioFeeer() {
	cases := []struct {
		name   string
		fee    float64
		min    string
		max    string
		big    string
		result string
		err    string
	}{
		{
			name:   "10 > 0.5, no min",
			fee:    0.5,
			min:    "0",
			big:    "10",
			result: "5",
		},
		{
			name:   "10 > 1.0, no min",
			fee:    1,
			min:    "0",
			big:    "10",
			result: "10",
		},
		{
			name:   "10 > 0.9, no min",
			fee:    0.9,
			min:    "0",
			big:    "10",
			result: "9",
		},
		{
			name:   "10 > 0.01, no min",
			fee:    0.01,
			min:    "0",
			big:    "10",
			result: "0",
		},
		{
			name:   "10 > 0.01, min=2",
			fee:    0.01,
			min:    "2",
			big:    "10",
			result: "2",
		},
		{
			name:   "10 > 0.01, min=20",
			fee:    0.01,
			min:    "20",
			big:    "10",
			result: "20",
		},
		{
			name:   "over max",
			fee:    0.99,
			min:    "1",
			max:    "10",
			big:    "20",
			result: "10",
		},
		{
			name:   "zero max",
			fee:    0.99,
			min:    "0",
			max:    "0",
			big:    "20",
			result: "0",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				min, err := NewBigFromString(c.min)
				t.NoError(err)

				var max Big = UnlimitedMaxFeeAmount
				if len(c.max) > 0 {
					i, err := NewBigFromString(c.max)
					t.NoError(err)
					max = i
				}

				big, err := NewBigFromString(c.big)
				t.NoError(err)

				receiver := MustAddress(util.UUID().String())
				fa := NewRatioFeeer(receiver, c.fee, min, max)
				t.NoError(fa.IsValid(nil))

				t.Equal(min, fa.Min())

				result, err := fa.Fee(big)
				if len(c.err) > 0 {
					t.Contains(err.Error(), c.err, "%d: %v; %v != %v", i, c.name, c.err, err)
				} else if err != nil {
					t.NoError(err, "%d: %v; %v != %v", i, c.name, c.err, err)
				} else {
					t.Equal(c.result, result.String(), "%d: %v; %v != %v", i, c.name, c.result, result.String())
				}
			},
		)
	}
}

func TestFeeer(t *testing.T) {
	suite.Run(t, new(testFeeer))
}

func testNilFeeerEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestEncode)

	t.enc = enc
	t.newObject = func() interface{} {
		return NewNilFeeer()
	}

	t.compare = func(a, b interface{}) {
		ca := a.(NilFeeer)
		cb := b.(NilFeeer)

		t.Equal(ca, cb)
	}

	return t
}

func testFixedFeeerEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestEncode)

	t.enc = enc
	t.newObject = func() interface{} {
		return NewFixedFeeer(
			MustAddress(util.UUID().String()),
			NewBig(33),
		)
	}

	t.compare = func(a, b interface{}) {
		ca := a.(FixedFeeer)
		cb := b.(FixedFeeer)

		t.Equal(ca, cb)
	}

	return t
}

func testRatioFeeerEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestEncode)

	t.enc = enc
	t.newObject = func() interface{} {
		return NewRatioFeeer(
			MustAddress(util.UUID().String()),
			0.777,
			NewBig(33),
			NewBig(34),
		)
	}

	t.compare = func(a, b interface{}) {
		ca := a.(RatioFeeer)
		cb := b.(RatioFeeer)

		t.Equal(ca, cb)
	}

	return t
}

func TestNilFeeerEncodeJSON(t *testing.T) {
	suite.Run(t, testNilFeeerEncode(jsonenc.NewEncoder()))
}

func TestFixedFeeerEncodeJSON(t *testing.T) {
	suite.Run(t, testFixedFeeerEncode(jsonenc.NewEncoder()))
}

func TestRatioFeeerEncodeJSON(t *testing.T) {
	suite.Run(t, testRatioFeeerEncode(jsonenc.NewEncoder()))
}

func TestNilFeeerEncodeBSON(t *testing.T) {
	suite.Run(t, testNilFeeerEncode(bsonenc.NewEncoder()))
}

func TestFixedFeeerEncodeBSON(t *testing.T) {
	suite.Run(t, testFixedFeeerEncode(bsonenc.NewEncoder()))
}

func TestRatioFeeerEncodeBSON(t *testing.T) {
	suite.Run(t, testRatioFeeerEncode(bsonenc.NewEncoder()))
}
