package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (ac *Account) unpack(enc encoder.Encoder, h valuehash.Hash, bad base.AddressDecoder, bks []byte) error {
	if a, err := bad.Encode(enc); err != nil {
		return err
	} else {
		ac.address = a
	}

	if hinter, err := enc.DecodeByHint(bks); err != nil {
		return err
	} else if k, ok := hinter.(Keys); !ok {
		return xerrors.Errorf("not Keys: %T", hinter)
	} else {
		ac.keys = k
	}

	ac.h = h

	return nil
}
