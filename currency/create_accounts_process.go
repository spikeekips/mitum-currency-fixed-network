package currency

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (CreateAccounts) Process(
	func(key string) (state.State, bool, error),
	func(valuehash.Hash, ...state.State) error,
) error {
	return nil
}

type CreateAccountsItemProcessor struct {
	cp   *CurrencyPool
	h    valuehash.Hash
	item CreateAccountsItem
	ns   state.State
	nb   map[CurrencyID]AmountState
}

func (opp *CreateAccountsItemProcessor) PreProcess(
	getState func(key string) (state.State, bool, error),
	_ func(valuehash.Hash, ...state.State) error,
) error {
	for i := range opp.item.Amounts() {
		am := opp.item.Amounts()[i]

		var policy CurrencyPolicy
		if opp.cp != nil {
			i, found := opp.cp.Policy(am.Currency())
			if !found {
				return errors.Errorf("currency not registered, %q", am.Currency())
			}
			policy = i
		}

		if am.Big().Compare(policy.NewAccountMinBalance()) < 0 {
			return errors.Errorf(
				"amount should be over minimum balance, %v < %v", am.Big(), policy.NewAccountMinBalance())
		}
	}

	target, err := opp.item.Address()
	if err != nil {
		return err
	}

	st, err := notExistsState(StateKeyAccount(target), "keys of target", getState)
	if err != nil {
		return err
	}
	opp.ns = st

	nb := map[CurrencyID]AmountState{}
	for i := range opp.item.Amounts() {
		am := opp.item.Amounts()[i]
		b, _, err := getState(StateKeyBalance(target, am.Currency()))
		if err != nil {
			return err
		}
		nb[am.Currency()] = NewAmountState(b, am.Currency())
	}

	opp.nb = nb

	return nil
}

func (opp *CreateAccountsItemProcessor) Process(
	_ func(key string) (state.State, bool, error),
	_ func(valuehash.Hash, ...state.State) error,
) ([]state.State, error) {
	nac, err := NewAccountFromKeys(opp.item.Keys())
	if err != nil {
		return nil, err
	}

	sts := make([]state.State, len(opp.item.Amounts())+1)
	st, err := SetStateAccountValue(opp.ns, nac)
	if err != nil {
		return nil, err
	}
	sts[0] = st

	for i := range opp.item.Amounts() {
		am := opp.item.Amounts()[i]
		sts[i+1] = opp.nb[am.Currency()].Add(am.Big())
	}

	return sts, nil
}

type CreateAccountsProcessor struct {
	cp *CurrencyPool
	CreateAccounts
	sb       map[CurrencyID]AmountState
	ns       []*CreateAccountsItemProcessor
	required map[CurrencyID][2]Big
}

func NewCreateAccountsProcessor(cp *CurrencyPool) GetNewProcessor {
	return func(op state.Processor) (state.Processor, error) {
		i, ok := op.(CreateAccounts)
		if !ok {
			return nil, errors.Errorf("not CreateAccounts, %T", op)
		}
		return &CreateAccountsProcessor{
			cp:             cp,
			CreateAccounts: i,
		}, nil
	}
}

func (opp *CreateAccountsProcessor) PreProcess(
	getState func(key string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) (state.Processor, error) {
	fact := opp.Fact().(CreateAccountsFact)

	if err := checkExistsState(StateKeyAccount(fact.sender), getState); err != nil {
		return nil, err
	}

	if required, err := opp.calculateItemsFee(); err != nil {
		return nil, operation.NewBaseReasonError("failed to calculate fee: %w", err)
	} else if sb, err := CheckEnoughBalance(fact.sender, required, getState); err != nil {
		return nil, err
	} else {
		opp.required = required
		opp.sb = sb
	}

	ns := make([]*CreateAccountsItemProcessor, len(fact.items))
	for i := range fact.items {
		c := &CreateAccountsItemProcessor{cp: opp.cp, h: opp.Hash(), item: fact.items[i]}
		if err := c.PreProcess(getState, setState); err != nil {
			return nil, operation.NewBaseReasonErrorFromError(err)
		}

		ns[i] = c
	}

	if err := checkFactSignsByState(fact.sender, opp.Signs(), getState); err != nil {
		return nil, operation.NewBaseReasonError("invalid signing: %w", err)
	}

	opp.ns = ns

	return opp, nil
}

func (opp *CreateAccountsProcessor) Process( // nolint:dupl
	getState func(key string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) error {
	fact := opp.Fact().(CreateAccountsFact)

	var sts []state.State // nolint:prealloc
	for i := range opp.ns {
		s, err := opp.ns[i].Process(getState, setState)
		if err != nil {
			return operation.NewBaseReasonError("failed to process create account item: %w", err)
		}
		sts = append(sts, s...)
	}

	for k := range opp.required {
		rq := opp.required[k]
		sts = append(sts, opp.sb[k].Sub(rq[0]).AddFee(rq[1]))
	}

	return setState(fact.Hash(), sts...)
}

func (opp *CreateAccountsProcessor) calculateItemsFee() (map[CurrencyID][2]Big, error) {
	fact := opp.Fact().(CreateAccountsFact)

	items := make([]AmountsItem, len(fact.items))
	for i := range fact.items {
		items[i] = fact.items[i]
	}

	return CalculateItemsFee(opp.cp, items)
}

func CalculateItemsFee(cp *CurrencyPool, items []AmountsItem) (map[CurrencyID][2]Big, error) {
	required := map[CurrencyID][2]Big{}

	for i := range items {
		it := items[i]

		for j := range it.Amounts() {
			am := it.Amounts()[j]

			rq := [2]Big{ZeroBig, ZeroBig}
			if k, found := required[am.Currency()]; found {
				rq = k
			}

			if cp == nil {
				required[am.Currency()] = [2]Big{rq[0].Add(am.Big()), rq[1]}

				continue
			}

			feeer, found := cp.Feeer(am.Currency())
			if !found {
				return nil, errors.Errorf("unknown currency id found, %q", am.Currency())
			}
			switch k, err := feeer.Fee(am.Big()); {
			case err != nil:
				return nil, err
			case !k.OverZero():
				required[am.Currency()] = [2]Big{rq[0].Add(am.Big()), rq[1]}
			default:
				required[am.Currency()] = [2]Big{rq[0].Add(am.Big()).Add(k), rq[1].Add(k)}
			}
		}
	}

	return required, nil
}

func CheckEnoughBalance(
	holder base.Address,
	required map[CurrencyID][2]Big,
	getState func(key string) (state.State, bool, error),
) (map[CurrencyID]AmountState, error) {
	sb := map[CurrencyID]AmountState{}

	for cid := range required {
		rq := required[cid]

		st, err := existsState(StateKeyBalance(holder, cid), "currency of holder", getState)
		if err != nil {
			return nil, err
		}

		am, err := StateBalanceValue(st)
		if err != nil {
			return nil, operation.NewBaseReasonError("insufficient balance of sender: %w", err)
		}

		if am.Big().Compare(rq[0]) < 0 {
			return nil, operation.NewBaseReasonError(
				"insufficient balance of sender, %s; %d !> %d", holder.String(), am.Big(), rq[0])
		}
		sb[cid] = NewAmountState(st, cid)
	}

	return sb, nil
}
