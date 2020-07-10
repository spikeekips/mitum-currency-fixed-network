package mc

import (
	"golang.org/x/xerrors"

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
	sender Address
	keys   Keys
	amount Amount
}

func NewCreateAccountFact(token []byte, sender Address, keys Keys, amount Amount) CreateAccountFact {
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

func (caf CreateAccountFact) Sender() Address {
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
}

func NewCreateAccount(fact CreateAccountFact, fs []operation.FactSign) (CreateAccount, error) {
	if bo, err := operation.NewBaseOperationFromFact(CreateAccountHint, fact, fs); err != nil {
		return CreateAccount{}, err
	} else {
		return CreateAccount{BaseOperation: bo}, nil
	}
}

func (ca CreateAccount) Hint() hint.Hint {
	return CreateAccountHint
}

func (ca CreateAccount) IsValid(networkID []byte) error {
	return operation.IsValidOperation(ca, networkID)
}

func (ca CreateAccount) ProcessOperation(
	getState func(key string) (state.StateUpdater, bool, error),
	setState func(state.StateUpdater) error,
) error {
	fact := ca.Fact().(CreateAccountFact)

	switch _, found, err := getState(StateKeyKeys(fact.sender)); {
	case err != nil:
		return err
	case !found:
		return state.IgnoreOperationProcessingError.Errorf("keys of sender account does not exist")
	}

	var newAddress Address
	if a, err := NewAddressFromKeys(fact.keys.Keys()); err != nil {
		return state.IgnoreOperationProcessingError.Wrap(err)
	} else {
		newAddress = a
	}

	var sstateBalance, nstate, nstateBalance state.StateUpdater
	switch st, found, err := getState(StateKeyKeys(newAddress)); {
	case err != nil:
		return err
	case found:
		return state.IgnoreOperationProcessingError.Errorf("keys of target account already exists")
	default:
		nstate = st
	}

	switch st, found, err := getState(StateKeyBalance(fact.sender)); {
	case err != nil:
		return err
	case !found:
		return state.IgnoreOperationProcessingError.Errorf("balance of sender account does not exist")
	default:
		sstateBalance = st
	}

	switch st, found, err := getState(StateKeyBalance(newAddress)); {
	case err != nil:
		return err
	case found:
		return state.IgnoreOperationProcessingError.Errorf("balance target account already exists")
	default:
		nstateBalance = st
	}

	if err := checkFactSignsByState(fact.sender, ca.Signs(), getState); err != nil {
		return state.IgnoreOperationProcessingError.Errorf("invalid signing: %w", err)
	}

	if b, err := StateAmountValue(sstateBalance); err != nil {
		return state.IgnoreOperationProcessingError.Wrap(err)
	} else {
		n := b.Sub(fact.amount)
		if err := n.IsValid(nil); err != nil {
			return state.IgnoreOperationProcessingError.Errorf("failed to sub amount from balance: %w", err)
		} else if err := SetStateAmountValue(sstateBalance, n); err != nil {
			return state.IgnoreOperationProcessingError.Wrap(err)
		}
	}

	if err := SetStateKeysValue(nstate, fact.keys); err != nil {
		return state.IgnoreOperationProcessingError.Wrap(err)
	}

	if err := SetStateAmountValue(nstateBalance, fact.amount); err != nil {
		return state.IgnoreOperationProcessingError.Wrap(err)
	}

	if err := setState(sstateBalance); err != nil {
		return err
	}
	if err := setState(nstate); err != nil {
		return err
	}
	if err := setState(nstateBalance); err != nil {
		return err
	}

	return nil
}
