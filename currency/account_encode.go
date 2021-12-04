package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (ac *Account) unpack(enc encoder.Encoder, h valuehash.Hash, bad base.AddressDecoder, bks []byte) error {
	a, err := bad.Encode(enc)
	if err != nil {
		return err
	}
	ac.address = a

	if len(bks) > 0 {
		i, err := DecodeAccountKeys(bks, enc)
		if err != nil {
			return err
		}

		ac.keys = i
	}

	ac.h = h

	return nil
}
