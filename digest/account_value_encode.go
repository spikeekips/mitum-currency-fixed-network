package digest

import (
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
)

func (va *AccountValue) unpack(enc encoder.Encoder, bac []byte, bb [][]byte, height, previousHeight base.Height) error {
	if bac != nil {
		if i, err := currency.DecodeAccount(enc, bac); err != nil {
			return err
		} else {
			va.ac = i
		}
	}

	balance := make([]currency.Amount, len(bb))
	for i := range bb {
		if j, err := currency.DecodeAmount(enc, bb[i]); err != nil {
			return err
		} else {
			balance[i] = j
		}
	}

	va.balance = balance
	va.height = height
	va.previousHeight = previousHeight

	return nil
}
