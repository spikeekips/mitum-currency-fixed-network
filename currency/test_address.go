//go:build test
// +build test

package currency

func MustAddress(s string) Address {
	a := NewAddress(s)
	if err := a.IsValid(nil); err != nil {
		panic(err)
	}

	return a
}
