package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
)

func (de *CurrencyDesign) unpack(enc encoder.Encoder, bam []byte, ga base.AddressDecoder, bpo []byte) error {
	i, err := DecodeAmount(enc, bam)
	if err != nil {
		return err
	}
	de.Amount = i

	a, err := ga.Encode(enc)
	if err != nil {
		return err
	}
	de.genesisAccount = a

	j, err := DecodeCurrencyPolicy(enc, bpo)
	if err != nil {
		return err
	}
	de.policy = j

	return nil
}
