package currency

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testCurrencyID struct {
	suite.Suite
}

func (t *testCurrencyID) TestFromString() {
	cases := []struct {
		name string
		s    string
		err  string
	}{
		{name: "ABC", s: "ABC"},
		{name: "contains digit", s: "A3C"},
		{name: "digit first", s: "3AC"},
		{name: "digit ends", s: "AC3"},
		{name: "too short", s: "AC", err: "invalid length"},
		{name: "too long", s: "ACAAAAAAAAAA", err: "invalid length"},
		{name: "abc: lowercase not allowed", s: "abc", err: "wrong currency id"},
		{name: "lowercase in the middle", s: "ABsC", err: "wrong currency id"},
		{name: "not allowed char first", s: ".ABC", err: "wrong currency id"},
		{name: "not allowed char ends", s: "ABC.", err: "wrong currency id"},
		{name: "tab", s: "AB\tC", err: "wrong currency id"},
		{name: "space", s: "AB C", err: "wrong currency id"},
		{name: "blanks first", s: " ABC", err: "wrong currency id"},
		{name: "blanks ends", s: "ABC ", err: "wrong currency id"},
		{name: "_", s: "A_BC"},
		{name: "-", s: "A-BC", err: "wrong currency id"},
		{name: ".", s: "A.BC"},
		{name: "!", s: "A!BC"},
		{name: "$", s: "A$BC"},
		{name: "*", s: "A*BC"},
		{name: "+", s: "A+BC"},
		{name: "@", s: "A@BC"},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				s := CurrencyID(c.s)
				err := s.IsValid(nil)
				if len(c.err) > 0 {
					var es string
					if err != nil {
						es = err.Error()
					}

					t.Contains(es, c.err, "%d: %v; %v != %v", i, c.name, c.err, es)
				} else if err != nil {
					t.NoError(err, "%d: %v; %v != %v", i, c.name, c.err, err)
				}
			},
		)
	}
}

func TestCurrencyID(t *testing.T) {
	suite.Run(t, new(testCurrencyID))
}
