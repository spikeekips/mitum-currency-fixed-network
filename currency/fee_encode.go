package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (fact *FeeOperationFact) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	token []byte,
	bam []byte,
) error {
	fact.h = h
	fact.token = token

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

	fact.amounts = amounts

	return nil
}

func (op *FeeOperation) unpack(enc encoder.Encoder, h valuehash.Hash, bfact []byte) error {
	if hinter, err := base.DecodeFact(bfact, enc); err != nil {
		return err
	} else if fact, ok := hinter.(FeeOperationFact); !ok {
		return xerrors.Errorf("not FeeOperationFact, %T", hinter)
	} else {
		op.fact = fact
	}

	op.h = h

	return nil
}
