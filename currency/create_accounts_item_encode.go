package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

func (it *BaseCreateAccountsItem) unpack(enc encoder.Encoder, ht hint.Hint, bks []byte, bam []byte) error {
	it.hint = ht

	if hinter, err := enc.Decode(bks); err != nil {
		return err
	} else if k, ok := hinter.(Keys); !ok {
		return xerrors.Errorf("not Keys: %T", hinter)
	} else {
		it.keys = k
	}

	ham, err := enc.DecodeSlice(bam)
	if err != nil {
		return err
	}

	amounts := make([]Amount, len(ham))
	for i := range ham {
		j, ok := ham[i].(Amount)
		if !ok {
			return util.WrongTypeError.Errorf("expected Amount, not %T", ham[i])
		}

		amounts[i] = j
	}

	it.amounts = amounts

	return nil
}
