package mc

import "github.com/spikeekips/mitum/util/encoder"

func (ca *Address) unpack(_ encoder.Encoder, s string) error {
	if a, err := NewAddress(s); err != nil {
		return err
	} else {
		*ca = a
	}

	return nil
}
