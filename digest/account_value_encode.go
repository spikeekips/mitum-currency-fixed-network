package digest

import (
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
)

func (va *AccountValue) unpack(enc encoder.Encoder, bac []byte, bb [][]byte, height, previousHeight base.Height) error {
	if bac != nil {
		i, err := currency.DecodeAccount(enc, bac)
		if err != nil {
			return err
		}
		va.ac = i
	}

	balance := make([]currency.Amount, len(bb))
	for i := range bb {
		j, err := currency.DecodeAmount(enc, bb[i])
		if err != nil {
			return err
		}
		balance[i] = j
	}

	va.balance = balance
	va.height = height
	va.previousHeight = previousHeight

	return nil
}
