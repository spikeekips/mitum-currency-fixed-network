package mc

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (tff *TransferFact) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	token []byte,
	bSender, bReceiver base.AddressDecoder,
	am Amount,
) error {
	var sender, receiver base.Address
	if a, err := bSender.Encode(enc); err != nil {
		return err
	} else {
		sender = a
	}

	if a, err := bReceiver.Encode(enc); err != nil {
		return err
	} else {
		receiver = a
	}

	tff.h = h
	tff.token = token
	tff.sender = sender
	tff.receiver = receiver
	tff.amount = am

	return nil
}
