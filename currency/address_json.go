package currency

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func (ca Address) MarshalText() ([]byte, error) {
	return ca.Bytes(), nil
}

func (ca *Address) UnpackJSON(b []byte, _ *jsonenc.Encoder) error {
	*ca = NewAddress(string(b))

	return nil
}
