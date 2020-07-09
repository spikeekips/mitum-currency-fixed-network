package mc

import (
	"math/big"

	"github.com/spikeekips/mitum/util"
)

func (a *Amount) UnmarshalJSON(b []byte) error {
	var i *big.Int
	if err := util.JSON.Unmarshal(b, &i); err != nil {
		return err
	}

	a.Int = i

	return nil
}
