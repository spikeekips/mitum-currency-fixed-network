package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

func (it *BaseCreateAccountsItem) unpack(enc encoder.Encoder, ht hint.Hint, bks []byte, bas [][]byte) error {
	it.hint = ht

	if hinter, err := enc.DecodeByHint(bks); err != nil {
		return err
	} else if k, ok := hinter.(Keys); !ok {
		return xerrors.Errorf("not Keys: %T", hinter)
	} else {
		it.keys = k
	}

	amounts := make([]Amount, len(bas))
	for i := range bas {
		j, err := DecodeAmount(enc, bas[i])
		if err != nil {
			return err
		}
		amounts[i] = j
	}

	it.amounts = amounts

	return nil
}
