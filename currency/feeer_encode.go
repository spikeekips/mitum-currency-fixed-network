package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

func (fa *FixedFeeer) unpack(enc encoder.Encoder, ht hint.Hint, brc base.AddressDecoder, am Big) error {
	fa.BaseHinter = hint.NewBaseHinter(ht)

	i, err := brc.Encode(enc)
	if err != nil {
		return err
	}
	fa.receiver = i

	fa.amount = am

	return nil
}

func (fa *RatioFeeer) unpack(
	enc encoder.Encoder,
	ht hint.Hint,
	brc base.AddressDecoder,
	ratio float64,
	min, max Big,
) error {
	fa.BaseHinter = hint.NewBaseHinter(ht)

	i, err := brc.Encode(enc)
	if err != nil {
		return err
	}
	fa.receiver = i

	fa.ratio = ratio
	fa.min = min
	fa.max = max

	return nil
}
