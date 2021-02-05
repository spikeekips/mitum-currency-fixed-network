package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

func (op CreateAccounts) Process(
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
			if i, found := opp.cp.Policy(am.Currency()); !found {
				return xerrors.Errorf("currency not registered, %q", am.Currency())
			} else {
				policy = i
			}
		}

		if am.Big().Compare(policy.NewAccountMinBalance()) < 0 {
			return xerrors.Errorf("amount should be over minimum balance, %v < %v", am.Big(), policy.NewAccountMinBalance())
		}
	}

	var target base.Address
	if a, err := opp.item.Address(); err != nil {
		return err
	} else {
		target = a
	}

	if st, err := notExistsState(StateKeyAccount(target), "keys of target", getState); err != nil {
		return err
	} else {
		opp.ns = st
	}

	nb := map[CurrencyID]AmountState{}
	for i := range opp.item.Amounts() {
		am := opp.item.Amounts()[i]
		if b, _, err := getState(StateKeyBalance(target, am.Currency())); err != nil {
			return err
		} else {
			nb[am.Currency()] = NewAmountState(b, am.Currency())
		}
	}

	opp.nb = nb

	return nil
}

func (opp *CreateAccountsItemProcessor) Process(
	_ func(key string) (state.State, bool, error),
	_ func(valuehash.Hash, ...state.State) error,
) ([]state.State, error) {
	var nac Account
	if ac, err := NewAccountFromKeys(opp.item.Keys()); err != nil {
		return nil, err
	} else {
		nac = ac
	}

	sts := make([]state.State, len(opp.item.Amounts())+1)
	if st, err := SetStateAccountValue(opp.ns, nac); err != nil {
		return nil, err
	} else {
		sts[0] = st
	}

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
		if i, ok := op.(CreateAccounts); !ok {
			return nil, xerrors.Errorf("not CreateAccounts, %T", op)
		} else {
			return &CreateAccountsProcessor{
				cp:             cp,
				CreateAccounts: i,
			}, nil
		}
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
		return nil, util.IgnoreError.Errorf("failed to calculate fee: %w", err)
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
			return nil, util.IgnoreError.Wrap(err)
		}

		ns[i] = c
	}

	if err := checkFactSignsByState(fact.sender, opp.Signs(), getState); err != nil {
		return nil, util.IgnoreError.Errorf("invalid signing: %w", err)
	}

	opp.ns = ns

	return opp, nil
}

func (opp *CreateAccountsProcessor) Process(
	getState func(key string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) error {
	fact := opp.Fact().(CreateAccountsFact)

	var sts []state.State // nolint:prealloc
	for i := range opp.ns {
		if s, err := opp.ns[i].Process(getState, setState); err != nil {
			return util.IgnoreError.Errorf("failed to process create account item: %w", err)
		} else {
			sts = append(sts, s...)
		}
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

			var rq [2]Big = [2]Big{ZeroBig, ZeroBig}
			if k, found := required[am.Currency()]; found {
				rq = k
			}

			if cp == nil {
				required[am.Currency()] = [2]Big{rq[0].Add(am.Big()), rq[1]}

				continue
			}

			if feeer, found := cp.Feeer(am.Currency()); !found {
				return nil, xerrors.Errorf("unknown currency id found, %q", am.Currency())
			} else {
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

		var st state.State
		if i, err := existsState(StateKeyBalance(holder, cid), "currency of holder", getState); err != nil {
			return nil, err
		} else {
			st = i
		}

		var am Amount
		if b, err := StateBalanceValue(st); err != nil {
			return nil, util.IgnoreError.Errorf("insufficient balance of sender: %w", err)
		} else {
			am = b
		}

		if am.Big().Compare(rq[0]) < 0 {
			return nil, util.IgnoreError.Errorf("insufficient balance of sender")
		} else {
			sb[cid] = NewAmountState(st, cid)
		}
	}

	return sb, nil
}
