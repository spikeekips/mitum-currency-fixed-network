package currency

import jsonenc "github.com/spikeekips/mitum/util/encoder/json"

func (a Amount) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(a.String())
}

func (a *Amount) UnmarshalJSON(b []byte) error {
	var s string
	if err := jsonenc.Unmarshal(b, &s); err != nil {
		return err
	}

	if i, err := NewAmountFromString(s); err != nil {
		return err
	} else {
		*a = i
	}

	return nil
}
