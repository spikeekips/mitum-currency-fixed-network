package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (CurrencyRegister) Process(
	func(string) (state.State, bool, error),
	func(valuehash.Hash, ...state.State) error,
) error {
	// NOTE Process is nil func
	return nil
}

type CurrencyRegisterProcessor struct {
	CurrencyRegister
	cp        *CurrencyPool
	pubs      []key.Publickey
	threshold base.Threshold
	ga        AmountState
	de        state.State
}

func NewCurrencyRegisterProcessor(cp *CurrencyPool, pubs []key.Publickey, threshold base.Threshold) GetNewProcessor {
	return func(op state.Processor) (state.Processor, error) {
		i, ok := op.(CurrencyRegister)
		if !ok {
			return nil, xerrors.Errorf("not CurrencyRegister, %T", op)
		}
		return &CurrencyRegisterProcessor{
			CurrencyRegister: i,
			cp:               cp,
			pubs:             pubs,
			threshold:        threshold,
		}, nil
	}
}

func (opp *CurrencyRegisterProcessor) PreProcess(
	getState func(string) (state.State, bool, error),
	_ func(valuehash.Hash, ...state.State) error,
) (state.Processor, error) {
	if len(opp.pubs) < 1 {
		return nil, xerrors.Errorf("empty publickeys for operation signs")
	} else if err := checkFactSignsByPubs(opp.pubs, opp.threshold, opp.Signs()); err != nil {
		return nil, err
	}

	item := opp.Fact().(CurrencyRegisterFact).currency

	if opp.cp != nil {
		if opp.cp.Exists(item.Currency()) {
			return nil, xerrors.Errorf("currency already registered, %q", item.Currency())
		}
	}

	if err := checkExistsState(StateKeyAccount(item.GenesisAccount()), getState); err != nil {
		return nil, xerrors.Errorf("genesis account not found: %w", err)
	}

	if receiver := item.Policy().Feeer().Receiver(); receiver != nil {
		if err := checkExistsState(StateKeyAccount(receiver), getState); err != nil {
			return nil, xerrors.Errorf("feeer receiver account not found: %w", err)
		}
	}

	switch st, found, err := getState(StateKeyCurrencyDesign(item.Currency())); {
	case err != nil:
		return nil, err
	case found:
		return nil, xerrors.Errorf("currency already registered, %q", item.Currency())
	default:
		opp.de = st
	}

	switch st, found, err := getState(StateKeyBalance(item.GenesisAccount(), item.Currency())); {
	case err != nil:
		return nil, err
	case found:
		return nil, xerrors.Errorf("genesis account has already the currency, %q", item.Currency())
	default:
		opp.ga = NewAmountState(st, item.Currency())
	}

	return opp, nil
}

func (opp *CurrencyRegisterProcessor) Process(
	_ func(string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) error {
	fact := opp.Fact().(CurrencyRegisterFact)

	sts := make([]state.State, 2)

	sts[0] = opp.ga.Add(fact.currency.Big())
	i, err := SetStateCurrencyDesignValue(opp.de, fact.currency)
	if err != nil {
		return err
	}
	sts[1] = i

	return setState(fact.Hash(), sts...)
}
