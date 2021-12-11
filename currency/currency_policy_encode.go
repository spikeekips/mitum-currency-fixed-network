package currency

import (
	"github.com/spikeekips/mitum/util/encoder"
)

func (po *CurrencyPolicy) unpack(enc encoder.Encoder, mn Big, bfe []byte) error {
	if err := encoder.Decode(bfe, enc, &po.feeer); err != nil {
		return err
	}

	po.newAccountMinBalance = mn

	return nil
}
