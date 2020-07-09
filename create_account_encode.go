package mc

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (caf *CreateAccountFact) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	tk []byte,
	sender Address,
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

	caf.h = h
	caf.token = tk
	caf.sender = sender
	caf.keys = keys
	caf.amount = am

	return nil
}
