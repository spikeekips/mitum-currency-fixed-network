package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
)

func (de *CurrencyDesign) unpack(enc encoder.Encoder, bam []byte, ga base.AddressDecoder, bpo []byte) error {
	if i, err := DecodeAmount(enc, bam); err != nil {
		return err
	} else {
		de.Amount = i
	}

	if i, err := ga.Encode(enc); err != nil {
		return err
	} else {
		de.genesisAccount = i
	}

	if i, err := DecodeCurrencyPolicy(enc, bpo); err != nil {
		return err
	} else {
		de.policy = i
	}

	return nil
}
