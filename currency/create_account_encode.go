package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (cai *CreateAccountItem) unpack(
	enc encoder.Encoder,
	bks []byte,
	am Amount,
) error {
	var keys Keys
	if hinter, err := enc.DecodeByHint(bks); err != nil {
		return err
	} else if k, ok := hinter.(Keys); !ok {
		return xerrors.Errorf("not Keys: %T", hinter)
	} else {
		keys = k
	}

	cai.keys = keys
	cai.amount = am

	return nil
}

func (caf *CreateAccountsFact) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	tk []byte,
	bSender base.AddressDecoder,
	its []CreateAccountItem,
) error {
	var sender base.Address
	if a, err := bSender.Encode(enc); err != nil {
		return err
	} else {
		sender = a
	}

	caf.h = h
	caf.token = tk
	caf.sender = sender
	caf.items = its

	return nil
}
