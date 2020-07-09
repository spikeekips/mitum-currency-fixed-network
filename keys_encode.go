package mc

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (ky *Key) unpack(enc encoder.Encoder, w uint, kd key.KeyDecoder) error {
	ky.w = w

	if k, err := kd.Encode(enc); err != nil {
		return err
	} else if pk, ok := k.(key.Publickey); !ok {
		return xerrors.Errorf("not key.Publickey; type=%T", k)
	} else {
		ky.k = pk
	}

	return nil
}

func (ks *Keys) unpack(enc encoder.Encoder, h valuehash.Hash, bkeys [][]byte, th uint) error {
	ks.h = h

	keys := make([]Key, len(bkeys))
	for i := range bkeys {
		if hinter, err := enc.DecodeByHint(bkeys[i]); err != nil {
			return err
		} else if k, ok := hinter.(Key); !ok {
			return xerrors.Errorf("not Key: %T", hinter)
		} else {
			keys[i] = k
		}
	}

	ks.keys = keys
	ks.threshold = th

	return nil
}
