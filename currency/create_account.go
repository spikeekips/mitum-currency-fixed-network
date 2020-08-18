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

type CreateAccountFact struct {
	h      valuehash.Hash
	token  []byte
	sender base.Address
	keys   Keys
	amount Amount
}

func NewCreateAccountFact(token []byte, sender base.Address, keys Keys, amount Amount) CreateAccountFact {
	caf := CreateAccountFact{
		token:  token,
		sender: sender,
		keys:   keys,
		amount: amount,
	}
	caf.h = valuehash.NewSHA256(caf.Bytes())

	return caf
}

func (caf CreateAccountFact) Hint() hint.Hint {
	return CreateAccountFactHint
}

func (caf CreateAccountFact) Hash() valuehash.Hash {
	return caf.h
}

func (caf CreateAccountFact) Bytes() []byte {
	return util.ConcatBytesSlice(
		caf.token,
		caf.sender.Bytes(),
		caf.keys.Bytes(),
		caf.amount.Bytes(),
	)
}

func (caf CreateAccountFact) IsValid([]byte) error {
	if len(caf.token) < 1 {
		return xerrors.Errorf("empty token for CreateAccountFact")
	}

	if err := isvalid.Check([]isvalid.IsValider{
		caf.h,
		caf.sender,
		caf.keys,
		caf.amount,
	}, nil, false); err != nil {
		return err
	}

	// TODO check minimum amount for create account and it should be managed by Policy

	return nil
}

func (caf CreateAccountFact) Token() []byte {
	return caf.token
}

func (caf CreateAccountFact) Sender() base.Address {
	return caf.sender
}

func (caf CreateAccountFact) Keys() Keys {
	return caf.keys
}

func (caf CreateAccountFact) Amount() Amount {
	return caf.amount
}

type CreateAccount struct {
	operation.BaseOperation
	Memo string
}

func NewCreateAccount(fact CreateAccountFact, fs []operation.FactSign, memo string) (CreateAccount, error) {
	if bo, err := operation.NewBaseOperationFromFact(CreateAccountHint, fact, fs); err != nil {
		return CreateAccount{}, err
	} else {
		ca := CreateAccount{BaseOperation: bo, Memo: memo}

		ca.BaseOperation = bo.SetHash(ca.GenerateHash())

		return ca, nil
	}
}

func (ca CreateAccount) Hint() hint.Hint {
	return CreateAccountHint
}

func (ca CreateAccount) IsValid(networkID []byte) error {
	if err := IsValidMemo(ca.Memo); err != nil {
		return err
	}

	return operation.IsValidOperation(ca, networkID)
}

func (ca CreateAccount) GenerateHash() valuehash.Hash {
	bs := make([][]byte, len(ca.Signs())+1)
	for i := range ca.Signs() {
		bs[i] = ca.Signs()[i].Bytes()
	}

	bs[len(bs)-1] = []byte(ca.Memo)

	e := util.ConcatBytesSlice(ca.Fact().Hash().Bytes(), util.ConcatBytesSlice(bs...))

	return valuehash.NewSHA256(e)
}

func (ca CreateAccount) AddFactSigns(fs ...operation.FactSign) (operation.FactSignUpdater, error) {
	if o, err := ca.BaseOperation.AddFactSigns(fs...); err != nil {
		return nil, err
	} else {
		ca.BaseOperation = o.(operation.BaseOperation)
	}

	ca.BaseOperation = ca.SetHash(ca.GenerateHash())

	return ca, nil
}

func (ca CreateAccount) Process(
	func(key string) (state.StateUpdater, bool, error),
	func(valuehash.Hash, ...state.StateUpdater) error,
) error {
	return nil
}

type CreateAccountProcessor struct {
	CreateAccount
	sb *AmountState
	na base.Address
	ns state.StateUpdater
	nb *AmountState
}

func (ca *CreateAccountProcessor) PreProcess(
	getState func(key string) (state.StateUpdater, bool, error),
	_ func(valuehash.Hash, ...state.StateUpdater) error,
) (state.Processor, error) {
	fact := ca.Fact().(CreateAccountFact)

	if fact.Amount().IsZero() {
		return nil, xerrors.Errorf("amount should be over zero")
	}

	if err := checkExistsAccountState(StateKeyKeys(fact.sender), getState); err != nil {
		return nil, err
	}

	if a, err := NewAddressFromKeys(fact.keys.Keys()); err != nil {
		return nil, state.IgnoreOperationProcessingError.Wrap(err)
	} else if st, err := notExistsAccountState(StateKeyKeys(a), "keys of target", getState); err != nil {
		return nil, err
	} else if b, err := notExistsAccountState(StateKeyBalance(a), "balance of target", getState); err != nil {
		return nil, err
	} else if ast, ok := b.(*AmountState); !ok {
		return nil, xerrors.Errorf("expected AmountState, but %T", st)
	} else {
		ca.na = a
		ca.ns = st
		ca.nb = ast
	}

	if st, err := existsAccountState(StateKeyBalance(fact.sender), "balance of sender", getState); err != nil {
		return nil, err
	} else if b, err := StateAmountValue(st); err != nil {
		return nil, state.IgnoreOperationProcessingError.Wrap(err)
	} else if b.Compare(fact.Amount()) < 0 {
		return nil, state.IgnoreOperationProcessingError.Errorf("insufficient balance of sender")
	} else if ast, ok := st.(*AmountState); !ok {
		return nil, xerrors.Errorf("expected AmountState, but %T", st)
	} else {
		ca.sb = ast
	}

	if err := checkFactSignsByState(fact.sender, ca.Signs(), getState); err != nil {
		return nil, state.IgnoreOperationProcessingError.Errorf("invalid signing: %w", err)
	}

	return ca, nil
}

func (ca *CreateAccountProcessor) Process(
	_ func(key string) (state.StateUpdater, bool, error),
	setState func(valuehash.Hash, ...state.StateUpdater) error,
) error {
	fact := ca.Fact().(CreateAccountFact)

	if ca.na == nil || ca.ns == nil || ca.sb == nil || ca.nb == nil {
		return xerrors.Errorf("PreProcess not executed")
	}

	if err := ca.sb.Sub(fact.amount); err != nil {
		return state.IgnoreOperationProcessingError.Errorf("failed to sub amount from balance: %w", err)
	}

	if err := SetStateKeysValue(ca.ns, fact.keys); err != nil {
		return state.IgnoreOperationProcessingError.Wrap(err)
	}

	if err := ca.nb.Add(fact.amount); err != nil {
		return state.IgnoreOperationProcessingError.Wrap(err)
	}

	return setState(ca.Hash(), ca.sb, ca.ns, ca.nb)
}
