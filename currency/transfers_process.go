package currency

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (Transfers) Process(
	func(key string) (state.State, bool, error),
	func(valuehash.Hash, ...state.State) error,
) error {
	// NOTE Process is nil func
	return nil
}

type TransfersItemProcessor struct {
	cp *CurrencyPool
	h  valuehash.Hash

	item TransfersItem

	rb map[CurrencyID]AmountState
}

func (opp *TransfersItemProcessor) PreProcess(
	getState func(key string) (state.State, bool, error),
	_ func(valuehash.Hash, ...state.State) error,
) error {
	if _, err := existsState(StateKeyAccount(opp.item.Receiver()), "receiver", getState); err != nil {
		return err
	}

	rb := map[CurrencyID]AmountState{}
	for i := range opp.item.Amounts() {
		am := opp.item.Amounts()[i]

		if opp.cp != nil {
			if !opp.cp.Exists(am.Currency()) {
				return errors.Errorf("currency not registered, %q", am.Currency())
			}
		}

		st, _, err := getState(StateKeyBalance(opp.item.Receiver(), am.Currency()))
		if err != nil {
			return err
		}
		rb[am.Currency()] = NewAmountState(st, am.Currency())
	}

	opp.rb = rb

	return nil
}

func (opp *TransfersItemProcessor) Process(
	_ func(key string) (state.State, bool, error),
	_ func(valuehash.Hash, ...state.State) error,
) ([]state.State, error) {
	sts := make([]state.State, len(opp.item.Amounts()))
	for i := range opp.item.Amounts() {
		am := opp.item.Amounts()[i]
		sts[i] = opp.rb[am.Currency()].Add(am.Big())
	}

	return sts, nil
}

type TransfersProcessor struct {
	cp *CurrencyPool
	Transfers
	sb       map[CurrencyID]AmountState
	rb       []*TransfersItemProcessor
	required map[CurrencyID][2]Big
}

func NewTransfersProcessor(cp *CurrencyPool) GetNewProcessor {
	return func(op state.Processor) (state.Processor, error) {
		i, ok := op.(Transfers)
		if !ok {
			return nil, errors.Errorf("not Transfers, %T", op)
		}
		return &TransfersProcessor{
			cp:        cp,
			Transfers: i,
		}, nil
	}
}

func (opp *TransfersProcessor) PreProcess(
	getState func(key string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) (state.Processor, error) {
	fact := opp.Fact().(TransfersFact)

	if err := checkExistsState(StateKeyAccount(fact.sender), getState); err != nil {
		return nil, err
	}

	if required, err := opp.calculateItemsFee(); err != nil {
		return nil, operation.NewBaseReasonErrorFromError(err)
	} else if sb, err := CheckEnoughBalance(fact.sender, required, getState); err != nil {
		return nil, err
	} else {
		opp.required = required
		opp.sb = sb
	}

	rb := make([]*TransfersItemProcessor, len(fact.items))
	for i := range fact.items {
		c := &TransfersItemProcessor{cp: opp.cp, h: opp.Hash(), item: fact.items[i]}
		if err := c.PreProcess(getState, setState); err != nil {
			return nil, operation.NewBaseReasonErrorFromError(err)
		}

		rb[i] = c
	}

	if err := checkFactSignsByState(fact.sender, opp.Signs(), getState); err != nil {
		return nil, errors.Wrap(err, "invalid signing")
	}

	opp.rb = rb

	return opp, nil
}

func (opp *TransfersProcessor) Process( // nolint:dupl
	getState func(key string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) error {
	fact := opp.Fact().(TransfersFact)

	var sts []state.State // nolint:prealloc
	for i := range opp.rb {
		s, err := opp.rb[i].Process(getState, setState)
		if err != nil {
			return operation.NewBaseReasonError("failed to process transfer item: %w", err)
		}
		sts = append(sts, s...)
	}

	for k := range opp.required {
		rq := opp.required[k]
		sts = append(sts, opp.sb[k].Sub(rq[0]).AddFee(rq[1]))
	}

	return setState(fact.Hash(), sts...)
}

func (opp *TransfersProcessor) calculateItemsFee() (map[CurrencyID][2]Big, error) {
	fact := opp.Fact().(TransfersFact)

	items := make([]AmountsItem, len(fact.items))
	for i := range fact.items {
		items[i] = fact.items[i]
	}

	return CalculateItemsFee(opp.cp, items)
}
