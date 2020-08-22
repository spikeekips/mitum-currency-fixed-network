package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (tff *TransferItem) unpack(enc encoder.Encoder, bReceiver base.AddressDecoder, am Amount) error {
	var receiver base.Address
	if a, err := bReceiver.Encode(enc); err != nil {
		return err
	} else {
		receiver = a
	}

	tff.receiver = receiver
	tff.amount = am

	return nil
}

func (tff *TransfersFact) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	token []byte,
	bSender base.AddressDecoder,
	items []TransferItem,
) error {
	var sender base.Address
	if a, err := bSender.Encode(enc); err != nil {
		return err
	} else {
		sender = a
	}

	tff.h = h
	tff.token = token
	tff.sender = sender
	tff.items = items

	return nil
}
