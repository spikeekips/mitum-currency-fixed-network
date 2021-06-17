package currency

import (
	"golang.org/x/xerrors"

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

	if hinter, err := enc.Decode(bks); err != nil {
		return err
	} else if k, ok := hinter.(Keys); !ok {
		return xerrors.Errorf("not Keys: %T", hinter)
	} else {
		ac.keys = k
	}

	ac.h = h

	return nil
}
