package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (fact *CreateAccountsFact) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	tk []byte,
	bSender base.AddressDecoder,
	bits [][]byte,
) error {
	var sender base.Address
	if a, err := bSender.Encode(enc); err != nil {
		return err
	} else {
		sender = a
	}

	its := make([]CreateAccountsItem, len(bits))
	for i := range bits {
		if j, err := DecodeCreateAccountsItem(enc, bits[i]); err != nil {
			return err
		} else {
			its[i] = j
		}
	}

	fact.h = h
	fact.token = tk
	fact.sender = sender
	fact.items = its

	return nil
}
