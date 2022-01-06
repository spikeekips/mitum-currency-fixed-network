package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
)

func checkFactSignsByPubs(pubs []key.Publickey, threshold base.Threshold, signs []base.FactSign) error {
	var signed uint
	for i := range signs {
		for j := range pubs {
			if signs[i].Signer().Equal(pubs[j]) {
				signed++

				break
			}
		}
	}

	if signed < threshold.Threshold {
		return operation.NewBaseReasonError("not enough suffrage signs")
	}

	return nil
}

func checkFactSignsByState(
	address base.Address,
	fs []base.FactSign,
	getState func(string) (state.State, bool, error),
) error {
	st, err := existsState(StateKeyAccount(address), "keys of account", getState)
	if err != nil {
		return err
	}
	keys, err := StateKeysValue(st)
	switch {
	case err != nil:
		return operation.NewBaseReasonErrorFromError(err)
	case keys == nil:
		return operation.NewBaseReasonError("empty keys found")
	}

	if err := checkThreshold(fs, keys); err != nil {
		return operation.NewBaseReasonErrorFromError(err)
	}

	return nil
}
