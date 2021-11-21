package currency

import (
	"github.com/pkg/errors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	CreateAccountsFactType = hint.Type("mitum-currency-create-accounts-operation-fact")
	CreateAccountsFactHint = hint.NewHint(CreateAccountsFactType, "v0.0.1")
	CreateAccountsType     = hint.Type("mitum-currency-create-accounts-operation")
	CreateAccountsHint     = hint.NewHint(CreateAccountsType, "v0.0.1")
	CreateAccountsHinter   = CreateAccounts{BaseOperation: operationHinter(CreateAccountsHint)}
)

var MaxCreateAccountsItems uint = 10

type AmountsItem interface {
	Amounts() []Amount
}

type CreateAccountsItem interface {
	hint.Hinter
	isvalid.IsValider
	AmountsItem
	Bytes() []byte
	Keys() Keys
	Address() (base.Address, error)
	Rebuild() CreateAccountsItem
}

type CreateAccountsFact struct {
	h      valuehash.Hash
	token  []byte
	sender base.Address
	items  []CreateAccountsItem
}

func NewCreateAccountsFact(token []byte, sender base.Address, items []CreateAccountsItem) CreateAccountsFact {
	fact := CreateAccountsFact{
		token:  token,
		sender: sender,
		items:  items,
	}
	fact.h = fact.GenerateHash()

	return fact
}

func (CreateAccountsFact) Hint() hint.Hint {
	return CreateAccountsFactHint
}

func (fact CreateAccountsFact) Hash() valuehash.Hash {
	return fact.h
}

func (fact CreateAccountsFact) GenerateHash() valuehash.Hash {
	return valuehash.NewSHA256(fact.Bytes())
}

func (fact CreateAccountsFact) Bytes() []byte {
	is := make([][]byte, len(fact.items))
	for i := range fact.items {
		is[i] = fact.items[i].Bytes()
	}

	return util.ConcatBytesSlice(
		fact.token,
		fact.sender.Bytes(),
		util.ConcatBytesSlice(is...),
	)
}

func (fact CreateAccountsFact) IsValid(b []byte) error {
	if err := IsValidOperationFact(fact, b); err != nil {
		return err
	}

	if n := len(fact.items); n < 1 {
		return errors.Errorf("empty items")
	} else if n > int(MaxCreateAccountsItems) {
		return errors.Errorf("items, %d over max, %d", n, MaxCreateAccountsItems)
	}

	if err := isvalid.Check([]isvalid.IsValider{fact.sender}, nil, false); err != nil {
		return err
	}

	foundKeys := map[string]struct{}{}
	for i := range fact.items {
		if err := fact.items[i].IsValid(nil); err != nil {
			return err
		}

		it := fact.items[i]
		k := it.Keys().Hash().String()
		if _, found := foundKeys[k]; found {
			return errors.Errorf("duplicated acocunt Keys found, %s", k)
		}

		switch a, err := it.Address(); {
		case err != nil:
			return err
		case fact.sender.Equal(a):
			return errors.Errorf("target address is same with sender, %q", fact.sender)
		default:
			foundKeys[k] = struct{}{}
		}
	}

	return nil
}

func (fact CreateAccountsFact) Token() []byte {
	return fact.token
}

func (fact CreateAccountsFact) Sender() base.Address {
	return fact.sender
}

func (fact CreateAccountsFact) Items() []CreateAccountsItem {
	return fact.items
}

func (fact CreateAccountsFact) Targets() ([]base.Address, error) {
	as := make([]base.Address, len(fact.items))
	for i := range fact.items {
		a, err := fact.items[i].Address()
		if err != nil {
			return nil, err
		}
		as[i] = a
	}

	return as, nil
}

func (fact CreateAccountsFact) Addresses() ([]base.Address, error) {
	as := make([]base.Address, len(fact.items)+1)

	tas, err := fact.Targets()
	if err != nil {
		return nil, err
	}
	copy(as, tas)

	as[len(fact.items)] = fact.Sender()

	return as, nil
}

func (fact CreateAccountsFact) Rebuild() CreateAccountsFact {
	items := make([]CreateAccountsItem, len(fact.items))
	for i := range fact.items {
		it := fact.items[i]
		items[i] = it.Rebuild()
	}

	fact.items = items
	fact.h = fact.GenerateHash()

	return fact
}

type CreateAccounts struct {
	BaseOperation
}

func NewCreateAccounts(fact CreateAccountsFact, fs []operation.FactSign, memo string) (CreateAccounts, error) {
	bo, err := NewBaseOperationFromFact(CreateAccountsHint, fact, fs, memo)
	if err != nil {
		return CreateAccounts{}, err
	}

	return CreateAccounts{BaseOperation: bo}, nil
}
