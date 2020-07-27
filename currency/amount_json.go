package currency

import (
	"github.com/spikeekips/mitum/util"
)

func (a Amount) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(a.String())
}

func (a *Amount) UnmarshalJSON(b []byte) error {
	var s string
	if err := util.JSON.Unmarshal(b, &s); err != nil {
		return err
	}

	if i, err := NewAmountFromString(s); err != nil {
		return err
	} else {
		*a = i
	}

	return nil
}
