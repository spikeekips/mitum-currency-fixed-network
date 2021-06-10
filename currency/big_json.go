package currency

import jsonenc "github.com/spikeekips/mitum/util/encoder/json"

func (a Big) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(a.String())
}

func (a *Big) UnmarshalJSON(b []byte) error {
	var s string
	if err := jsonenc.Unmarshal(b, &s); err != nil {
		return err
	}

	i, err := NewBigFromString(s)
	if err != nil {
		return err
	}
	*a = i

	return nil
}
