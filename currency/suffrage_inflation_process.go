package currency

import (
	"github.com/pkg/errors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (SuffrageInflation) Process(
	func(string) (state.State, bool, error),
	func(valuehash.Hash, ...state.State) error,
) error {
	// NOTE Process is nil func
	return nil
}

type SuffrageInflationProcessor struct {
	SuffrageInflation
	cp        *CurrencyPool
	pubs      []key.Publickey
	threshold base.Threshold
	ast       map[string]AmountState
	dst       map[CurrencyID]state.State
	dc        map[CurrencyID]CurrencyDesign
}

func NewSuffrageInflationProcessor(cp *CurrencyPool, pubs []key.Publickey, threshold base.Threshold) GetNewProcessor {
	return func(op state.Processor) (state.Processor, error) {
		i, ok := op.(SuffrageInflation)
		if !ok {
			return nil, errors.Errorf("not SuffrageInflation, %T", op)
		}

		return &SuffrageInflationProcessor{
			SuffrageInflation: i,
			cp:                cp,
			pubs:              pubs,
			threshold:         threshold,
		}, nil
	}
}

func (opp *SuffrageInflationProcessor) PreProcess(
	getState func(string) (state.State, bool, error),
	_ func(valuehash.Hash, ...state.State) error,
) (state.Processor, error) {
	if len(opp.pubs) < 1 {
		return nil, errors.Errorf("empty publickeys for operation signs")
	} else if err := checkFactSignsByPubs(opp.pubs, opp.threshold, opp.Signs()); err != nil {
		return nil, err
	}

	items := opp.Fact().(SuffrageInflationFact).items

	ast := map[string]AmountState{}
	dst := map[CurrencyID]state.State{}
	dc := map[CurrencyID]CurrencyDesign{}
	for i := range items {
		item := items[i]
		cid := item.amount.Currency()
		st, found := opp.cp.State(cid)
		if !found {
			return nil, operation.NewBaseReasonError("unknown currency, %q for SuffrageInflation", cid)
		}
		dst[cid] = st

		if err := checkExistsState(StateKeyAccount(item.receiver), getState); err != nil {
			return nil, errors.Wrap(err, "unknown receiver of SuffrageInflation")
		}

		aid := StateKeyBalance(item.receiver, item.amount.Currency())
		if _, found := ast[aid]; !found {
			bst, _, err := getState(StateKeyBalance(item.receiver, cid))
			if err != nil {
				return nil, operation.NewBaseReasonError("inflation receiver account not found: %w", err)
			}

			ast[aid] = NewAmountState(bst, cid)
		}

		dc[cid], _ = opp.cp.Get(cid)
	}

	opp.ast = ast
	opp.dst = dst
	opp.dc = dc

	return opp, nil
}

func (opp *SuffrageInflationProcessor) Process(
	_ func(string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) error {
	items := opp.Fact().(SuffrageInflationFact).items

	sts := make([]state.State, len(opp.ast)+len(opp.dst))

	inc := map[CurrencyID]Big{}
	for i := range items {
		item := items[i]
		aid := StateKeyBalance(item.receiver, item.amount.Currency())
		opp.ast[aid] = opp.ast[aid].Add(item.amount.Big())
		inc[item.amount.Currency()] = item.amount.Big()
	}

	var i int
	for k := range opp.ast {
		sts[i] = opp.ast[k]
		i++
	}

	for cid := range inc {
		dc, err := opp.dc[cid].AddAggregate(inc[cid])
		if err != nil {
			return err
		}

		j, err := SetStateCurrencyDesignValue(opp.dst[cid], dc)
		if err != nil {
			return err
		}

		sts[i] = j
		i++
	}

	return setState(opp.Fact().Hash(), sts...)
}
