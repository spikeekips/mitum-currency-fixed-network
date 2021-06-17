package currency

import (
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (ky *Key) unpack(enc encoder.Encoder, w uint, kd key.PublickeyDecoder) error {
	ky.w = w

	k, err := kd.Encode(enc)
	if err != nil {
		return err
	}
	ky.k = k

	return nil
}

func (ks *Keys) unpack(enc encoder.Encoder, h valuehash.Hash, bks []byte, th uint) error {
	ks.h = h

	hks, err := enc.DecodeSlice(bks)
	if err != nil {
		return err
	}

	keys := make([]Key, len(hks))
	for i := range hks {
		j, ok := hks[i].(Key)
		if !ok {
			return util.WrongTypeError.Errorf("expected Key, not %T", hks[i])
		}

		keys[i] = j
	}

	ks.keys = keys
	ks.threshold = th

	return nil
}
