package currency

import "github.com/spikeekips/mitum/util/hint"

func TypedString(ht hint.Hinter, s string) string {
	return hint.NewTypedString(ht.Hint().Type(), s).String()
}
