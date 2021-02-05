package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (fact *KeyUpdaterFact) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	token []byte,
	btarget base.AddressDecoder,
	bks []byte,
	cr string,
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

	fact.h = h
	fact.token = token
	fact.target = target
	fact.keys = keys
	fact.currency = CurrencyID(cr)

	return nil
}
