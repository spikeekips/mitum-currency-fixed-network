package mc

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (gaf *GenesisAccountFact) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	tk []byte,
	genesisNodeKey key.PublickeyDecoder,
	bks []byte,
	am Amount,
) error {
	var gkey key.Publickey
	if k, err := genesisNodeKey.Encode(enc); err != nil {
		return err
	} else {
		gkey = k
	}

	var keys Keys
	if hinter, err := enc.DecodeByHint(bks); err != nil {
		return err
	} else if k, ok := hinter.(Keys); !ok {
		return xerrors.Errorf("not Keys: %T", hinter)
	} else {
		keys = k
	}

	gaf.h = h
	gaf.token = tk
	gaf.genesisNodeKey = gkey
	gaf.keys = keys
	gaf.amount = am

	return nil
}
