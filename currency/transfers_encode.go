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

	a, err := bReceiver.Encode(enc)
	if err != nil {
		return err
	}
	it.receiver = a

	am := make([]Amount, len(bam))
	for i := range bam {
		j, err := DecodeAmount(enc, bam[i])
		if err != nil {
			return err
		}
		am[i] = j
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
	sender, err := bSender.Encode(enc)
	if err != nil {
		return err
	}

	items := make([]TransfersItem, len(bitems))
	for i := range bitems {
		j, err := DecodeTransfersItem(enc, bitems[i])
		if err != nil {
			return err
		}
		items[i] = j
	}

	fact.h = h
	fact.token = token
	fact.sender = sender
	fact.items = items

	return nil
}
