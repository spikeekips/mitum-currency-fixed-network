package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (it *BaseTransfersItem) unpack(
	enc encoder.Encoder,
	ht hint.Hint,
	bReceiver base.AddressDecoder,
	bam [][]byte,
) error {
	it.hint = ht

	if a, err := bReceiver.Encode(enc); err != nil {
		return err
	} else {
		it.receiver = a
	}

	am := make([]Amount, len(bam))
	for i := range bam {
		if j, err := DecodeAmount(enc, bam[i]); err != nil {
			return err
		} else {
			am[i] = j
		}
	}

	it.amounts = am

	return nil
}

func (fact *TransfersFact) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	token []byte,
	bSender base.AddressDecoder,
	bitems [][]byte,
) error {
	var sender base.Address
	if a, err := bSender.Encode(enc); err != nil {
		return err
	} else {
		sender = a
	}

	items := make([]TransfersItem, len(bitems))
	for i := range bitems {
		if j, err := DecodeTransfersItem(enc, bitems[i]); err != nil {
			return err
		} else {
			items[i] = j
		}
	}

	fact.h = h
	fact.token = token
	fact.sender = sender
	fact.items = items

	return nil
}
