package currency

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (fact *CurrencyRegisterFact) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	token []byte,
	bcr []byte,
) error {
	fact.h = h
	fact.token = token

	i, err := DecodeCurrencyDesign(enc, bcr)
	if err != nil {
		return err
	}
	fact.currency = i

	return nil
}
