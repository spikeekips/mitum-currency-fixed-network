package currency

import (
	"github.com/spikeekips/mitum/util/encoder"
)

func (po *CurrencyPolicy) unpack(enc encoder.Encoder, mn Big, bfe []byte) error {
	i, err := DecodeFeeer(enc, bfe)
	if err != nil {
		return err
	}
	po.feeer = i

	po.newAccountMinBalance = mn

	return nil
}
