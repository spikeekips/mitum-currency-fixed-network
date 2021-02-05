package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
)

func (fa *FixedFeeer) unpack(enc encoder.Encoder, brc base.AddressDecoder, am Big) error {
	if i, err := brc.Encode(enc); err != nil {
		return err
	} else {
		fa.receiver = i
	}

	fa.amount = am

	return nil
}

func (fa *RatioFeeer) unpack(enc encoder.Encoder, brc base.AddressDecoder, ratio float64, min, max Big) error {
	if i, err := brc.Encode(enc); err != nil {
		return err
	} else {
		fa.receiver = i
	}

	fa.ratio = ratio
	fa.min = min
	fa.max = max

	return nil
}
