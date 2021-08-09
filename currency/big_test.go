package currency

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/util"
)

type testBig struct {
	suite.Suite
}

func (t *testBig) TestFromString() {
	cases := []struct {
		name string
		s    string
		big  string
		err  string
	}{
		{
			name: "10",
			s:    "10",
			big:  "10",
		},
		{
			name: "-1",
			s:    "-1",
			big:  "-1",
		},
		{
			name: "over max uint64",
			s:    "922337203685477580792233720368547758079223372036854775807",
			big:  "922337203685477580792233720368547758079223372036854775807",
		},
		{
			name: "lower lowest int64",
			s:    "-922337203685477580892233720368547758089223372036854775808",
			big:  "-922337203685477580892233720368547758089223372036854775808",
		},
		{
			name: "alphabet",
			s:    "showme",
			err:  "not proper Big string",
		},
		{
			name: "hex",
			s:    "0x50",
			err:  "not proper Big string",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				big, err := NewBigFromString(c.s)
				if len(c.err) > 0 {
					t.Contains(err.Error(), c.err, "%d: %v; %v != %v", i, c.name, c.err, err)
				} else if err != nil {
					t.NoError(err, "%d: %v; %v != %v", i, c.name, c.err, err)
				} else {
					t.Equal(c.big, big.String(), "%d: %v; %v != %v", i, c.name, c.big, big.String())
				}
			},
		)
	}
}

func (t *testBig) TestAdd() {
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
				a := MustBigFromString(c.a)
				b := MustBigFromString(c.b)
				result := a.Add(b)

				t.Equal(c.result, result.String(), "%d: %v; %v != %v", i, c.name, c.result, result.String())
			},
		)
	}
}

func (t *testBig) TestSub() {
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
				a := MustBigFromString(c.a)
				b := MustBigFromString(c.b)
				result := a.Sub(b)

				t.Equal(c.result, result.String(), "%d: %v; %v != %v", i, c.name, c.result, result.String())
			},
		)
	}
}

func (t *testBig) TestMul() {
	cases := []struct {
		name   string
		a      string
		b      interface{}
		result string
	}{
		{
			name:   "3*7",
			a:      "3",
			b:      int64(7),
			result: "21",
		},
		{
			name:   "3*(-7)",
			a:      "3",
			b:      int64(-7),
			result: "-21",
		},
		{
			name:   "3*(-0.5)",
			a:      "3",
			b:      float64(-0.5),
			result: "-1",
		},
		{
			name:   "10*(0.5)",
			a:      "10",
			b:      float64(0.5),
			result: "5",
		},
		{
			name:   "10*(0.33)",
			a:      "10",
			b:      float64(0.33),
			result: "3",
		},
		{
			name:   "over max uint64",
			a:      "922337203685477580792233720368547758079223372036854775817",
			b:      int64(10),
			result: "9223372036854775807922337203685477580792233720368547758170",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				a := MustBigFromString(c.a)

				var result Big
				switch reflect.TypeOf(c.b).Kind() {
				case reflect.Int64:
					result = a.MulInt64(c.b.(int64))
				case reflect.Float64:
					result = a.MulFloat64(c.b.(float64))
				default:
					t.NoError(errors.Errorf("unsupported type"))
				}

				t.Equal(c.result, result.String(), "%d: %v; %v != %v", i, c.name, c.result, result.String())
			},
		)
	}
}

func (t *testBig) TestDiv() {
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
				a := MustBigFromString(c.a)
				b := MustBigFromString(c.b)
				result := a.Div(b)

				t.Equal(c.result, result.String(), "%d: %v; %v != %v", i, c.name, c.result, result.String())
			},
		)
	}
}

func (t *testBig) TestCmp() {
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
				a := MustBigFromString(c.a)
				b := MustBigFromString(c.b)
				result := a.Compare(b)

				t.Equal(c.result, result, "%d: %v; %v != %v", i, c.name, c.result, result)
			},
		)
	}
}

func (t *testBig) TestEqual() {
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
				a := MustBigFromString(c.a)
				b := MustBigFromString(c.b)
				result := a.Equal(b)

				t.Equal(c.result, result, "%d: %v; %v != %v", i, c.name, c.result, result)
			},
		)
	}
}

func (t *testBig) TestIsValid() {
	cases := []struct {
		name string
		big  string
		err  string
	}{
		{
			name: "10",
			big:  "10",
		},
		{
			name: "-1",
			big:  "-1",
		},
		{
			name: "over max uint64",
			big:  "922337203685477580792233720368547758079223372036854775807",
		},
		{
			name: "lower lowest int64",
			big:  "-922337203685477580892233720368547758089223372036854775808",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				big := MustBigFromString(c.big)
				err := big.IsValid(nil)
				if len(c.err) > 0 {
					t.Contains(err.Error(), c.err, "%d: %v; %v != %v", i, c.name, c.err, err)
				} else if err != nil {
					t.NoError(err, "%d: %v; %v != %v", i, c.name, c.err, err)
				} else {
					t.Equal(c.big, big.String(), "%d: %v; %v != %v", i, c.name, c.big, big.String())
				}
			},
		)
	}
}

func (t *testBig) testEncoding(
	mfunc func(interface{}) ([]byte, error),
	ufunc func([]byte) (interface{}, error),
) {
	cases := []struct {
		name string
		big  string
	}{
		{
			name: "10",
			big:  "10",
		},
		{
			name: "-1",
			big:  "-1",
		},
		{
			name: "over max uint64",
			big:  "922337203685477580792233720368547758079223372036854775807",
		},
		{
			name: "lower lowest int64",
			big:  "-922337203685477580892233720368547758089223372036854775808",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				big := MustBigFromString(c.big)
				b, err := mfunc(big)
				if err != nil {
					t.NoError(err, "%d: %v", i, c.name)
				}

				if o, err := ufunc(b); err != nil {
					t.NoError(err, "%d: %v", i, c.name)
				} else if am, ok := o.(Big); !ok {
					t.NoError(errors.Errorf("the returned, %T is not Big type", o), "%d: %v", i, c.name)
				} else {
					t.True(big.Equal(am), "%d: %v; %v != %v", i, c.name, big.String(), am.String())
				}
			},
		)
	}
}

func (t *testBig) TestEncodingJSON() {
	t.testEncoding(
		util.JSON.Marshal,
		func(b []byte) (interface{}, error) {
			var am Big
			if err := util.JSON.Unmarshal(b, &am); err != nil {
				return nil, err
			}

			return am, nil
		},
	)
}

func (t *testBig) TestEncodingBSON() {
	t.testEncoding(
		func(i interface{}) ([]byte, error) {
			am := struct {
				A Big
			}{A: i.(Big)}

			return bson.Marshal(am)
		},
		func(b []byte) (interface{}, error) {
			var am struct {
				A Big
			}

			if err := bson.Unmarshal(b, &am); err != nil {
				return nil, err
			}

			return am.A, nil
		},
	)
}

func TestBig(t *testing.T) {
	suite.Run(t, new(testBig))
}
