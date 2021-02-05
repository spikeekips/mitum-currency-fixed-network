package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
)

func checkFactSignsByPubs(pubs []key.Publickey, threshold base.Threshold, signs []operation.FactSign) error {
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
		return util.IgnoreError.Errorf("not enough suffrage signs")
	}

	return nil
}

func checkFactSignsByState(
	address base.Address,
	fs []operation.FactSign,
	getState func(key string) (state.State, bool, error),
) error {
	var keys Keys
	if st, err := existsState(StateKeyAccount(address), "keys of account", getState); err != nil {
		return err
	} else {
		if ks, err := StateKeysValue(st); err != nil {
			return util.IgnoreError.Wrap(err)
		} else {
			keys = ks
		}
	}

	if err := checkThreshold(fs, keys); err != nil {
		return util.IgnoreError.Wrap(err)
	}

	return nil
}
