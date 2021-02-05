package currency

import (
	"github.com/spikeekips/mitum/util/encoder"
)

func (po *CurrencyPolicy) unpack(enc encoder.Encoder, mn Big, bfe []byte) error {
	if i, err := DecodeFeeer(enc, bfe); err != nil {
		return err
	} else {
		po.feeer = i
	}

	po.newAccountMinBalance = mn

	return nil
}
