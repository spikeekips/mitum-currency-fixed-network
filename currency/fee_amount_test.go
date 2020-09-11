package currency

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testFeeAmount struct {
	suite.Suite
}

func (t *testFeeAmount) TestFixedFeeAmount() {
	cases := []struct {
		name   string
		fee    string
		amount string
		result string
		err    string
	}{
		{
			name:   "10 > 10",
			fee:    "10",
			amount: "10",
			result: "10",
		},
		{
			name:   "10 > 5",
			fee:    "5",
			amount: "10",
			result: "5",
		},
		{
			name:   "5 > 10",
			fee:    "10",
			amount: "5",
			result: "10",
		},
		{
			name:   "5 > 0",
			fee:    "0",
			amount: "5",
			result: "0",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				fee, err := NewAmountFromString(c.fee)
				t.NoError(err)

				amount, err := NewAmountFromString(c.amount)
				t.NoError(err)

				fa := NewFixedFeeAmount(fee)
				t.Equal(fee, fa.Min())

				result, err := fa.Fee(amount)
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

func (t *testFeeAmount) TestRatioFeeAmount() {
	cases := []struct {
		name   string
		fee    float64
		min    string
		amount string
		result string
		err    string
	}{
		{
			name:   "10 > 0.5, no min",
			fee:    0.5,
			min:    "0",
			amount: "10",
			result: "5",
		},
		{
			name:   "10 > 1.0, no min",
			fee:    1,
			min:    "0",
			amount: "10",
			result: "10",
		},
		{
			name:   "10 > 0.9, no min",
			fee:    0.9,
			min:    "0",
			amount: "10",
			result: "9",
		},
		{
			name:   "10 > 0.01, no min",
			fee:    0.01,
			min:    "0",
			amount: "10",
			result: "0",
		},
		{
			name:   "10 > 0.01, min=2",
			fee:    0.01,
			min:    "2",
			amount: "10",
			result: "2",
		},
		{
			name:   "10 > 0.01, min=20",
			fee:    0.01,
			min:    "20",
			amount: "10",
			result: "20",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				min, err := NewAmountFromString(c.min)
				t.NoError(err)

				amount, err := NewAmountFromString(c.amount)
				t.NoError(err)

				fa, err := NewRatioFeeAmount(c.fee, min)
				t.NoError(err)

				t.Equal(min, fa.Min())

				result, err := fa.Fee(amount)
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

func TestFeeAmount(t *testing.T) {
	suite.Run(t, new(testFeeAmount))
}
