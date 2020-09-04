package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	CreateAccountFactType = hint.MustNewType(0xa0, 0x05, "mitum-currency-create-account-operation-fact")
	CreateAccountFactHint = hint.MustHint(CreateAccountFactType, "0.0.1")
	CreateAccountType     = hint.MustNewType(0xa0, 0x06, "mitum-currency-create-account-operation")
	CreateAccountHint     = hint.MustHint(CreateAccountType, "0.0.1")
)

var (
	// TODO check minimum amount for create account and it should be managed by Policy
	MinAccountBalance     Amount = NewAmount(1)
	maxCreateAccountItems uint   = 10
)

type CreateAccountItem struct {
	keys   Keys
	amount Amount
}

func NewCreateAccountItem(keys Keys, amount Amount) CreateAccountItem {
	return CreateAccountItem{
		keys:   keys,
		amount: amount,
	}
}

func (cai CreateAccountItem) Bytes() []byte {
	return util.ConcatBytesSlice(
		cai.keys.Bytes(),
		cai.amount.Bytes(),
	)
}

func (cai CreateAccountItem) IsValid([]byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		cai.keys,
		cai.amount,
	}, nil, false); err != nil {
		return err
	}

	if cai.amount.IsZero() {
		return xerrors.Errorf("amount should be over zero")
	}

	return nil
}

func (cai CreateAccountItem) Keys() Keys {
	return cai.keys
}

func (cai CreateAccountItem) Amount() Amount {
	return cai.amount
}

func (cai CreateAccountItem) Address() (base.Address, error) {
	return NewAddressFromKeys(cai.keys)
}

type CreateAccountsFact struct {
	h      valuehash.Hash
	token  []byte
	sender base.Address
	items  []CreateAccountItem
}

func NewCreateAccountsFact(token []byte, sender base.Address, items []CreateAccountItem) CreateAccountsFact {
	caf := CreateAccountsFact{
		token:  token,
		sender: sender,
		items:  items,
	}
	caf.h = valuehash.NewSHA256(caf.Bytes())

	return caf
}

func (caf CreateAccountsFact) Hint() hint.Hint {
	return CreateAccountFactHint
}

func (caf CreateAccountsFact) Hash() valuehash.Hash {
	return caf.h
}

func (caf CreateAccountsFact) Bytes() []byte {
	is := make([][]byte, len(caf.items))
	for i := range caf.items {
		is[i] = caf.items[i].Bytes()
	}

	return util.ConcatBytesSlice(
		caf.token,
		caf.sender.Bytes(),
		util.ConcatBytesSlice(is...),
	)
}

func (caf CreateAccountsFact) IsValid([]byte) error {
	if len(caf.token) < 1 {
		return xerrors.Errorf("empty token for CreateAccountFact")
	} else if n := len(caf.items); n < 1 {
		return xerrors.Errorf("empty items")
	} else if n > int(maxCreateAccountItems) {
		return xerrors.Errorf("items, %d over max, %d", n, maxCreateAccountItems)
	}

	if err := isvalid.Check([]isvalid.IsValider{
		caf.h,
		caf.sender,
	}, nil, false); err != nil {
		return err
	}

	foundKeys := map[string]struct{}{}
	for i := range caf.items {
		if err := caf.items[i].IsValid(nil); err != nil {
			return err
		}

		it := caf.items[i]
		k := it.keys.Hash().String()
		if _, found := foundKeys[k]; found {
			return xerrors.Errorf("duplicated acocunt Keys found, %s", k)
		}

		switch a, err := it.Address(); {
		case err != nil:
			return err
		case caf.sender.Equal(a):
			return xerrors.Errorf("target address is same with sender, %q", caf.sender)
		default:
			foundKeys[k] = struct{}{}
		}
	}

	return nil
}

func (caf CreateAccountsFact) Token() []byte {
	return caf.token
}

func (caf CreateAccountsFact) Sender() base.Address {
	return caf.sender
}

func (caf CreateAccountsFact) Items() []CreateAccountItem {
	return caf.items
}

func (caf CreateAccountsFact) Amount() Amount {
	a := NewAmount(0)
	for i := range caf.items {
		a = a.Add(caf.items[i].Amount())
	}

	return a
}

func (caf CreateAccountsFact) Addresses() ([]base.Address, error) {
	as := make([]base.Address, len(caf.items))
	for i := range caf.items {
		if a, err := caf.items[i].Address(); err != nil {
			return nil, err
		} else {
			as[i] = a
		}
	}

	return as, nil
}

type CreateAccounts struct {
	operation.BaseOperation
	Memo string
}

func NewCreateAccounts(fact CreateAccountsFact, fs []operation.FactSign, memo string) (CreateAccounts, error) {
	if bo, err := operation.NewBaseOperationFromFact(CreateAccountHint, fact, fs); err != nil {
		return CreateAccounts{}, err
	} else {
		ca := CreateAccounts{BaseOperation: bo, Memo: memo}

		ca.BaseOperation = bo.SetHash(ca.GenerateHash())

		return ca, nil
	}
}

func (ca CreateAccounts) Hint() hint.Hint {
	return CreateAccountHint
}

func (ca CreateAccounts) IsValid(networkID []byte) error {
	if err := IsValidMemo(ca.Memo); err != nil {
		return err
	}

	return operation.IsValidOperation(ca, networkID)
}

func (ca CreateAccounts) GenerateHash() valuehash.Hash {
	bs := make([][]byte, len(ca.Signs())+1)
	for i := range ca.Signs() {
		bs[i] = ca.Signs()[i].Bytes()
	}

	bs[len(bs)-1] = []byte(ca.Memo)

	e := util.ConcatBytesSlice(ca.Fact().Hash().Bytes(), util.ConcatBytesSlice(bs...))

	return valuehash.NewSHA256(e)
}

func (ca CreateAccounts) AddFactSigns(fs ...operation.FactSign) (operation.FactSignUpdater, error) {
	if o, err := ca.BaseOperation.AddFactSigns(fs...); err != nil {
		return nil, err
	} else {
		ca.BaseOperation = o.(operation.BaseOperation)
	}

	ca.BaseOperation = ca.SetHash(ca.GenerateHash())

	return ca, nil
}

func (ca CreateAccounts) Process(
	func(key string) (state.State, bool, error),
	func(valuehash.Hash, ...state.State) error,
) error {
	return nil
}

type CreateAccountItemProcessor struct {
	h    valuehash.Hash
	fact CreateAccountItem
	ns   state.State
	nb   AmountState
}

func (ca *CreateAccountItemProcessor) PreProcess(
	getState func(key string) (state.State, bool, error),
	_ func(valuehash.Hash, ...state.State) error,
) error {
	if ca.fact.amount.Compare(MinAccountBalance) < 0 {
		return xerrors.Errorf("amount should be over minimum balance, %v", MinAccountBalance)
	}

	if a, err := ca.fact.Address(); err != nil {
		return err
	} else if st, err := notExistsAccountState(StateKeyKeys(a), "keys of target", getState); err != nil {
		return err
	} else if b, err := notExistsAccountState(StateKeyBalance(a), "balance of target", getState); err != nil {
		return err
	} else if ast, ok := b.(AmountState); !ok {
		return xerrors.Errorf("expected AmountState, but %T", st)
	} else {
		ca.ns = st
		ca.nb = ast
	}

	return nil
}

func (ca *CreateAccountItemProcessor) Process(
	_ func(key string) (state.State, bool, error),
	_ func(valuehash.Hash, ...state.State) error,
) ([]state.State, error) {
	sts := make([]state.State, 2)
	if st, err := SetStateKeysValue(ca.ns, ca.fact.keys); err != nil {
		return nil, err
	} else {
		sts[0] = st
	}

	sts[1] = ca.nb.Add(ca.fact.amount)

	return sts, nil
}

type CreateAccountsProcessor struct {
	CreateAccounts
	sb AmountState
	ns []*CreateAccountItemProcessor
}

func (ca *CreateAccountsProcessor) PreProcess(
	getState func(key string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) (state.Processor, error) {
	fact := ca.Fact().(CreateAccountsFact)

	if err := checkExistsAccountState(StateKeyKeys(fact.sender), getState); err != nil {
		return nil, err
	}

	if st, err := existsAccountState(StateKeyBalance(fact.sender), "balance of sender", getState); err != nil {
		return nil, err
	} else if b, err := StateAmountValue(st); err != nil {
		return nil, state.IgnoreOperationProcessingError.Wrap(err)
	} else if b.Compare(fact.Amount()) < 0 {
		return nil, state.IgnoreOperationProcessingError.Errorf("insufficient balance of sender")
	} else if ast, ok := st.(AmountState); !ok {
		return nil, xerrors.Errorf("expected AmountState, but %T", st)
	} else {
		ca.sb = ast
	}

	ns := make([]*CreateAccountItemProcessor, len(fact.items))
	for i := range fact.items {
		c := &CreateAccountItemProcessor{h: ca.Hash(), fact: fact.items[i]}
		if err := c.PreProcess(getState, setState); err != nil {
			return nil, state.IgnoreOperationProcessingError.Wrap(err)
		}

		ns[i] = c
	}

	if err := checkFactSignsByState(fact.sender, ca.Signs(), getState); err != nil {
		return nil, state.IgnoreOperationProcessingError.Errorf("invalid signing: %w", err)
	}

	ca.ns = ns

	return ca, nil
}

func (ca *CreateAccountsProcessor) Process(
	getState func(key string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) error {
	fact := ca.Fact().(CreateAccountsFact)

	sts := make([]state.State, len(ca.ns)*2+1)
	for i := range ca.ns {
		if s, err := ca.ns[i].Process(getState, setState); err != nil {
			return state.IgnoreOperationProcessingError.Errorf("failed to process create account item: %w", err)
		} else {
			sts[i*2] = s[0]
			sts[i*2+1] = s[1]
		}
	}

	sts[len(sts)-1] = ca.sb.Sub(fact.Amount())

	return setState(fact.Hash(), sts...)
}
