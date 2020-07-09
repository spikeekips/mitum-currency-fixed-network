// +build test

package mc

func MustAddress(s string) Address {
	a, err := NewAddress(s)
	if err != nil {
		panic(err)
	}

	return a
}
