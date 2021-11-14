package currency

import (
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (op GenesisCurrencies) Process(
	getState func(key string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) error {
	fact := op.Fact().(GenesisCurrenciesFact)

	newAddress, err := fact.Address()
	if err != nil {
		return operation.NewBaseReasonErrorFromError(err)
	}

	ns, err := notExistsState(StateKeyAccount(newAddress), "key of genesis", getState)
	if err != nil {
		return err
	}

	gas := map[CurrencyID]state.State{}
	sts := map[CurrencyID]state.State{}
	for i := range fact.cs {
		c := fact.cs[i]

		st, err := notExistsState(StateKeyCurrencyDesign(c.Currency()), "currency", getState)
		if err != nil {
			return err
		}
		sts[c.Currency()] = st

		st, err = notExistsState(StateKeyBalance(newAddress, c.Currency()), "balance of genesis", getState)
		if err != nil {
			return err
		}
		gas[c.Currency()] = NewAmountState(st, c.Currency())
	}

	var states []state.State
	if ac, err := NewAccountFromKeys(fact.keys); err != nil {
		return err
	} else if st, err := SetStateAccountValue(ns, ac); err != nil {
		return operation.NewBaseReasonErrorFromError(err)
	} else {
		states = append(states, st)
	}

	for i := range fact.cs {
		c := fact.cs[i]
		am := NewAmount(c.Big(), c.Currency())
		if gst, err := SetStateBalanceValue(gas[c.Currency()], am); err != nil {
			return err
		} else if dst, err := SetStateCurrencyDesignValue(sts[c.Currency()], c); err != nil {
			return err
		} else {
			states = append(states, gst, dst)
		}

		sts, err := createZeroAccount(c.Currency(), getState)
		if err != nil {
			return err
		}

		states = append(states, sts...)
	}

	return setState(fact.Hash(), states...)
}
