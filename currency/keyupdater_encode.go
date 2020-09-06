package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (ft *KeyUpdaterFact) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	token []byte,
	btarget base.AddressDecoder,
	bks []byte,
) error {
	var target base.Address
	if a, err := btarget.Encode(enc); err != nil {
		return err
	} else {
		target = a
	}

	var keys Keys
	if hinter, err := enc.DecodeByHint(bks); err != nil {
		return err
	} else if k, ok := hinter.(Keys); !ok {
		return xerrors.Errorf("not Keys: %T", hinter)
	} else {
		keys = k
	}

	ft.h = h
	ft.token = token
	ft.target = target
	ft.keys = keys

	return nil
}
