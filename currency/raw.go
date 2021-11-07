package currency

import "github.com/spikeekips/mitum/util/hint"

type rawHinted interface {
	hint.Hinter
	Raw() string
}

func RawTypeString(k rawHinted) string {
	return k.Raw() + "-" + k.Hint().Type().String()
}
