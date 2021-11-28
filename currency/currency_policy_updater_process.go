package currency

import (
	"github.com/pkg/errors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (CurrencyPolicyUpdater) Process(
	func(string) (state.State, bool, error),
	func(valuehash.Hash, ...state.State) error,
) error {
	// NOTE Process is nil func
	return nil
}

type CurrencyPolicyUpdaterProcessor struct {
	CurrencyPolicyUpdater
	cp        *CurrencyPool
	pubs      []key.Publickey
	threshold base.Threshold
	st        state.State
	de        CurrencyDesign
}

func NewCurrencyPolicyUpdaterProcessor(
	cp *CurrencyPool,
	pubs []key.Publickey,
	threshold base.Threshold,
) GetNewProcessor {
	return func(op state.Processor) (state.Processor, error) {
		i, ok := op.(CurrencyPolicyUpdater)
		if !ok {
			return nil, errors.Errorf("not CurrencyPolicyUpdater, %T", op)
		}
		return &CurrencyPolicyUpdaterProcessor{
			CurrencyPolicyUpdater: i,
			cp:                    cp,
			pubs:                  pubs,
			threshold:             threshold,
		}, nil
	}
}

func (opp *CurrencyPolicyUpdaterProcessor) PreProcess(
	getState func(string) (state.State, bool, error),
	_ func(valuehash.Hash, ...state.State) error,
) (state.Processor, error) {
	if len(opp.pubs) < 1 {
		return nil, operation.NewBaseReasonError("empty publickeys for operation signs")
	} else if err := checkFactSignsByPubs(opp.pubs, opp.threshold, opp.Signs()); err != nil {
		return nil, err
	}

	fact := opp.Fact().(CurrencyPolicyUpdaterFact)

	if opp.cp != nil {
		i, found := opp.cp.State(fact.Currency())
		if !found {
			return nil, operation.NewBaseReasonError("unknown currency, %q found", fact.Currency())
		}
		opp.st = i
		opp.de, _ = opp.cp.Get(fact.Currency())
	}

	if receiver := fact.Policy().Feeer().Receiver(); receiver != nil {
		if err := checkExistsState(StateKeyAccount(receiver), getState); err != nil {
			return nil, errors.Wrap(err, "feeer receiver account not found")
		}
	}

	return opp, nil
}

func (opp *CurrencyPolicyUpdaterProcessor) Process(
	_ func(string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) error {
	fact := opp.Fact().(CurrencyPolicyUpdaterFact)

	i, err := SetStateCurrencyDesignValue(opp.st, opp.de.SetPolicy(fact.Policy()))
	if err != nil {
		return err
	}
	return setState(fact.Hash(), i)
}
