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
	getState func(key string) (state.StateUpdater, bool, error),
	setState func(valuehash.Hash, ...state.StateUpdater) error,
) error {
	fact := ca.Fact().(CreateAccountFact)

	if _, err := existsAccountState(StateKeyKeys(fact.sender), "keys of sender", getState); err != nil {
		return err
	}

	var newAddress Address
	if a, err := NewAddressFromKeys(fact.keys.Keys()); err != nil {
		return state.IgnoreOperationProcessingError.Wrap(err)
	} else {
		newAddress = a
	}

	var err error
	var sBalance, nstate, nBalance state.StateUpdater
	{
		if nstate, err = notExistsAccountState(StateKeyKeys(newAddress), "keys of target", getState); err != nil {
			return err
		}

		if sBalance, err = existsAccountState(StateKeyBalance(fact.sender), "balance of sender", getState); err != nil {
			return err
		}

		if nBalance, err = notExistsAccountState(
			StateKeyBalance(newAddress), "balance of target", getState); err != nil {
			return err
		}
	}

	if err := checkFactSignsByState(fact.sender, ca.Signs(), getState); err != nil {
		return state.IgnoreOperationProcessingError.Errorf("invalid signing: %w", err)
	}

	if b, err := StateAmountValue(sBalance); err != nil {
		return state.IgnoreOperationProcessingError.Wrap(err)
	} else {
		n := b.Sub(fact.amount)
		if err := n.IsValid(nil); err != nil {
			return state.IgnoreOperationProcessingError.Errorf("failed to sub amount from balance: %w", err)
		} else if err := SetStateAmountValue(sBalance, n); err != nil {
			return state.IgnoreOperationProcessingError.Wrap(err)
		}
	}

	if err := SetStateKeysValue(nstate, fact.keys); err != nil {
		return state.IgnoreOperationProcessingError.Wrap(err)
	}

	if err := SetStateAmountValue(nBalance, fact.amount); err != nil {
		return state.IgnoreOperationProcessingError.Wrap(err)
	}

	return setState(ca.Hash(), sBalance, nstate, nBalance)
}
