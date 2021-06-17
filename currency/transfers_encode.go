package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (it *BaseTransfersItem) unpack(
	enc encoder.Encoder,
	ht hint.Hint,
	bReceiver base.AddressDecoder,
	bam []byte,
) error {
	it.hint = ht

	a, err := bReceiver.Encode(enc)
	if err != nil {
		return err
	}
	it.receiver = a

	ham, err := enc.DecodeSlice(bam)
	if err != nil {
		return err
	}

	am := make([]Amount, len(ham))
	for i := range ham {
		j, ok := ham[i].(Amount)
		if !ok {
			return util.WrongTypeError.Errorf("expected Amount, not %T", ham[i])
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
	bits []byte,
) error {
	sender, err := bSender.Encode(enc)
	if err != nil {
		return err
	}

	hits, err := enc.DecodeSlice(bits)
	if err != nil {
		return err
	}

	items := make([]TransfersItem, len(hits))
	for i := range hits {
		j, ok := hits[i].(TransfersItem)
		if !ok {
			return util.WrongTypeError.Errorf("expected TransfersItem, not %T", hits[i])
		}

		items[i] = j
	}

	fact.h = h
	fact.token = token
	fact.sender = sender
	fact.items = items

	return nil
}
