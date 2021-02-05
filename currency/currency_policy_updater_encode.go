package currency

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (fact *CurrencyPolicyUpdaterFact) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	token []byte,
	scid string,
	bpo []byte,
) error {
	fact.h = h
	fact.token = token

	fact.cid = CurrencyID(scid)

	if i, err := DecodeCurrencyPolicy(enc, bpo); err != nil {
		return err
	} else {
		fact.policy = i
	}

	return nil
}
