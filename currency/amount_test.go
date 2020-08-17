package currency

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
)

type testAmount struct {
	suite.Suite
}

func (t *testAmount) TestFromString() {
	cases := []struct {
		name   string
		s      string
		amount string
		err    string
	}{
		{
			name:   "10",
			s:      "10",
			amount: "10",
		},
		{
			name:   "-1",
			s:      "-1",
			amount: "-1",
		},
		{
			name:   "over max uint64",
			s:      "922337203685477580792233720368547758079223372036854775807",
			amount: "922337203685477580792233720368547758079223372036854775807",
		},
		{
			name:   "lower lowest int64",
			s:      "-922337203685477580892233720368547758089223372036854775808",
			amount: "-922337203685477580892233720368547758089223372036854775808",
		},
		{
			name: "alphabet",
			s:    "showme",
			err:  "not proper Amount string",
		},
		{
			name: "hex",
			s:    "0x50",
			err:  "not proper Amount string",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				amount, err := NewAmountFromString(c.s)
				if len(c.err) > 0 {
					t.Contains(err.Error(), c.err, "%d: %v; %v != %v", i, c.name, c.err, err)
				} else if err != nil {
					t.NoError(err, "%d: %v; %v != %v", i, c.name, c.err, err)
				} else {
					t.Equal(c.amount, amount.String(), "%d: %v; %v != %v", i, c.name, c.amount, amount.String())
				}
			},
		)
	}
}

func (t *testAmount) TestAdd() {
	cases := []struct {
		name   string
		a      string
		b      string
		result string
	}{
		{
			name:   "3+7",
			a:      "3",
			b:      "7",
			result: "10",
		},
		{
			name:   "3-7",
			a:      "3",
			b:      "-7",
			result: "-4",
		},
		{
			name:   "over max uint64",
			a:      "922337203685477580792233720368547758079223372036854775807",
			b:      "10",
			result: "922337203685477580792233720368547758079223372036854775817",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				a := MustAmountFromString(c.a)
				b := MustAmountFromString(c.b)
				result := a.Add(b)

				t.Equal(c.result, result.String(), "%d: %v; %v != %v", i, c.name, c.result, result.String())
			},
		)
	}
}

func (t *testAmount) TestSub() {
	cases := []struct {
		name   string
		a      string
		b      string
		result string
	}{
		{
			name:   "3-7",
			a:      "3",
			b:      "7",
			result: "-4",
		},
		{
			name:   "3-(-7)",
			a:      "3",
			b:      "-7",
			result: "10",
		},
		{
			name:   "over max uint64",
			a:      "922337203685477580792233720368547758079223372036854775817",
			b:      "10",
			result: "922337203685477580792233720368547758079223372036854775807",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				a := MustAmountFromString(c.a)
				b := MustAmountFromString(c.b)
				result := a.Sub(b)

				t.Equal(c.result, result.String(), "%d: %v; %v != %v", i, c.name, c.result, result.String())
			},
		)
	}
}

func (t *testAmount) TestMul() {
	cases := []struct {
		name   string
		a      string
		b      string
		result string
	}{
		{
			name:   "3*7",
			a:      "3",
			b:      "7",
			result: "21",
		},
		{
			name:   "3*(-7)",
			a:      "3",
			b:      "-7",
			result: "-21",
		},
		{
			name:   "over max uint64",
			a:      "922337203685477580792233720368547758079223372036854775817",
			b:      "10",
			result: "9223372036854775807922337203685477580792233720368547758170",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				a := MustAmountFromString(c.a)
				b := MustAmountFromString(c.b)
				result := a.Mul(b)

				t.Equal(c.result, result.String(), "%d: %v; %v != %v", i, c.name, c.result, result.String())
			},
		)
	}
}

func (t *testAmount) TestDiv() {
	cases := []struct {
		name   string
		a      string
		b      string
		result string
	}{
		{
			name:   "10/3",
			a:      "10",
			b:      "3",
			result: "3",
		},
		{
			name:   "3/7",
			a:      "3",
			b:      "7",
			result: "0",
		},
		{
			name:   "3/(-7)",
			a:      "3",
			b:      "-7",
			result: "0",
		},
		{
			name:   "over max uint64",
			a:      "922337203685477580792233720368547758079223372036854775817",
			b:      "10",
			result: "92233720368547758079223372036854775807922337203685477581",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				a := MustAmountFromString(c.a)
				b := MustAmountFromString(c.b)
				result := a.Div(b)

				t.Equal(c.result, result.String(), "%d: %v; %v != %v", i, c.name, c.result, result.String())
			},
		)
	}
}

func (t *testAmount) TestCmp() {
	cases := []struct {
		name   string
		a      string
		b      string
		result int
	}{
		{
			name:   "3 > 3",
			a:      "3",
			b:      "3",
			result: 0,
		},
		{
			name:   "10 > 3",
			a:      "10",
			b:      "3",
			result: 1,
		},
		{
			name:   "3> 7",
			a:      "3",
			b:      "7",
			result: -1,
		},
		{
			name:   "3 > -7",
			a:      "3",
			b:      "-7",
			result: 1,
		},
		{
			name:   "over max uint64",
			a:      "922337203685477580792233720368547758079223372036854775817",
			b:      "10",
			result: 1,
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				a := MustAmountFromString(c.a)
				b := MustAmountFromString(c.b)
				result := a.Compare(b)

				t.Equal(c.result, result, "%d: %v; %v != %v", i, c.name, c.result, result)
			},
		)
	}
}

func (t *testAmount) TestEqual() {
	cases := []struct {
		name   string
		a      string
		b      string
		result bool
	}{
		{
			name:   "3 == 3",
			a:      "3",
			b:      "3",
			result: true,
		},
		{
			name:   "10 == 3",
			a:      "10",
			b:      "3",
			result: false,
		},
		{
			name:   "3 == 7",
			a:      "3",
			b:      "7",
			result: false,
		},
		{
			name:   "3 == -7",
			a:      "3",
			b:      "-7",
			result: false,
		},
		{
			name:   "over max uint64, equal",
			a:      "922337203685477580792233720368547758079223372036854775817",
			b:      "922337203685477580792233720368547758079223372036854775817",
			result: true,
		},
		{
			name:   "over max uint64, not equal",
			a:      "922337203685477580792233720368547758079223372036854775817",
			b:      "1922337203685477580792233720368547758079223372036854775817",
			result: false,
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				a := MustAmountFromString(c.a)
				b := MustAmountFromString(c.b)
				result := a.Equal(b)

				t.Equal(c.result, result, "%d: %v; %v != %v", i, c.name, c.result, result)
			},
		)
	}
}

func (t *testAmount) TestIsValid() {
	cases := []struct {
		name   string
		amount string
		err    string
	}{
		{
			name:   "10",
			amount: "10",
		},
		{
			name:   "-1",
			amount: "-1",
			err:    "under zero",
		},
		{
			name:   "over max uint64",
			amount: "922337203685477580792233720368547758079223372036854775807",
		},
		{
			name:   "lower lowest int64",
			amount: "-922337203685477580892233720368547758089223372036854775808",
			err:    "under zero",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				amount := MustAmountFromString(c.amount)
				err := amount.IsValid(nil)
				if len(c.err) > 0 {
					t.Contains(err.Error(), c.err, "%d: %v; %v != %v", i, c.name, c.err, err)
				} else if err != nil {
					t.NoError(err, "%d: %v; %v != %v", i, c.name, c.err, err)
				} else {
					t.Equal(c.amount, amount.String(), "%d: %v; %v != %v", i, c.name, c.amount, amount.String())
				}
			},
		)
	}
}

func (t *testAmount) testEncoding(
	mfunc func(interface{}) ([]byte, error),
	ufunc func([]byte) (interface{}, error),
) {
	cases := []struct {
		name   string
		amount string
	}{
		{
			name:   "10",
			amount: "10",
		},
		{
			name:   "-1",
			amount: "-1",
		},
		{
			name:   "over max uint64",
			amount: "922337203685477580792233720368547758079223372036854775807",
		},
		{
			name:   "lower lowest int64",
			amount: "-922337203685477580892233720368547758089223372036854775808",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				amount := MustAmountFromString(c.amount)
				b, err := mfunc(amount)
				if err != nil {
					t.NoError(err, "%d: %v", i, c.name)
				}

				if o, err := ufunc(b); err != nil {
					t.NoError(err, "%d: %v", i, c.name)
				} else if am, ok := o.(Amount); !ok {
					t.NoError(xerrors.Errorf("the returned, %T is not Amount type", o), "%d: %v", i, c.name)
				} else {
					t.True(amount.Equal(am), "%d: %v; %v != %v", i, c.name, amount.String(), am.String())
				}
			},
		)
	}
}

func (t *testAmount) TestEncodingJSON() {
	t.testEncoding(
		util.JSON.Marshal,
		func(b []byte) (interface{}, error) {
			var am Amount
			if err := util.JSON.Unmarshal(b, &am); err != nil {
				return nil, err
			}

			return am, nil
		},
	)
}

func (t *testAmount) TestEncodingBSON() {
	t.testEncoding(
		func(i interface{}) ([]byte, error) {
			am := struct {
				A Amount
			}{A: i.(Amount)}

			return bson.Marshal(am)
		},
		func(b []byte) (interface{}, error) {
			var am struct {
				A Amount
			}

			if err := bson.Unmarshal(b, &am); err != nil {
				return nil, err
			}

			return am.A, nil
		},
	)
}

func TestAmount(t *testing.T) {
	suite.Run(t, new(testAmount))
}
