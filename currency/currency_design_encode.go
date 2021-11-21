package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
)

func (de *CurrencyDesign) unpack(enc encoder.Encoder, am Amount, ga base.AddressDecoder, bpo []byte, ag Big) error {
	de.Amount = am

	a, err := ga.Encode(enc)
	if err != nil {
		return err
	}
	de.genesisAccount = a

	j, err := DecodeCurrencyPolicy(bpo, enc)
	if err != nil {
		return err
	}
	de.policy = j
	de.aggregate = ag

	return nil
}
