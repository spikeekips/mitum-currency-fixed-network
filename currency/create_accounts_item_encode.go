package currency

import (
	"github.com/pkg/errors"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

func (it *BaseCreateAccountsItem) unpack(enc encoder.Encoder, bks []byte, bam []byte) error {
	if hinter, err := enc.Decode(bks); err != nil {
		return err
	} else if k, ok := hinter.(AccountKeys); !ok {
		return errors.Errorf("not Keys: %T", hinter)
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
