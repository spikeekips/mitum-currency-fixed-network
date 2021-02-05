package currency

import (
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

func (fact *GenesisCurrenciesFact) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	tk []byte,
	genesisNodeKey key.PublickeyDecoder,
	bks []byte,
	bcs [][]byte,
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

	fact.h = h
	fact.token = tk
	fact.genesisNodeKey = gkey
	fact.keys = keys

	cs := make([]CurrencyDesign, len(bcs))
	for i := range bcs {
		if j, err := DecodeCurrencyDesign(enc, bcs[i]); err != nil {
			return err
		} else {
			cs[i] = j
		}
	}

	fact.cs = cs

	return nil
}
