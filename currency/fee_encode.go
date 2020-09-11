package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (ft *FeeOperationFact) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	token []byte,
	fa string,
	breceiver base.AddressDecoder,
	fee Amount,
) error {
	var receiver base.Address
	if a, err := breceiver.Encode(enc); err != nil {
		return err
	} else {
		receiver = a
	}

	ft.h = h
	ft.token = token
	ft.fa = fa
	ft.receiver = receiver
	ft.fee = fee

	return nil
}

func (op *FeeOperation) unpack(enc encoder.Encoder, h valuehash.Hash, bfact []byte) error {
	if hinter, err := base.DecodeFact(enc, bfact); err != nil {
		return err
	} else if fact, ok := hinter.(FeeOperationFact); !ok {
		return xerrors.Errorf("not FeeOperationFact, %T", hinter)
	} else {
		op.fact = fact
	}

	op.h = h

	return nil
}
