package currency

import (
	"golang.org/x/xerrors"

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

func (fact CreateAccountsFact) Hint() hint.Hint {
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

func (fact CreateAccountsFact) IsValid([]byte) error {
	if len(fact.token) < 1 {
		return xerrors.Errorf("empty token for CreateAccountsFact")
	} else if n := len(fact.items); n < 1 {
		return xerrors.Errorf("empty items")
	} else if n > int(MaxCreateAccountsItems) {
		return xerrors.Errorf("items, %d over max, %d", n, MaxCreateAccountsItems)
	}

	if err := isvalid.Check([]isvalid.IsValider{
		fact.h,
		fact.sender,
	}, nil, false); err != nil {
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
			return xerrors.Errorf("duplicated acocunt Keys found, %s", k)
		}

		switch a, err := it.Address(); {
		case err != nil:
			return err
		case fact.sender.Equal(a):
			return xerrors.Errorf("target address is same with sender, %q", fact.sender)
		default:
			foundKeys[k] = struct{}{}
		}
	}

	if !fact.h.Equal(fact.GenerateHash()) {
		return isvalid.InvalidError.Errorf("wrong Fact hash")
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
		if a, err := fact.items[i].Address(); err != nil {
			return nil, err
		} else {
			as[i] = a
		}
	}

	return as, nil
}

func (fact CreateAccountsFact) Addresses() ([]base.Address, error) {
	as := make([]base.Address, len(fact.items)+1)

	if tas, err := fact.Targets(); err != nil {
		return nil, err
	} else {
		copy(as, tas)
	}

	as[len(fact.items)] = fact.Sender()

	return as, nil
}

func (fact CreateAccountsFact) Rebulild() CreateAccountsFact {
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
	operation.BaseOperation
	Memo string
}

func NewCreateAccounts(fact CreateAccountsFact, fs []operation.FactSign, memo string) (CreateAccounts, error) {
	if bo, err := operation.NewBaseOperationFromFact(CreateAccountsHint, fact, fs); err != nil {
		return CreateAccounts{}, err
	} else {
		op := CreateAccounts{BaseOperation: bo, Memo: memo}

		op.BaseOperation = bo.SetHash(op.GenerateHash())

		return op, nil
	}
}

func (op CreateAccounts) Hint() hint.Hint {
	return CreateAccountsHint
}

func (op CreateAccounts) IsValid(networkID []byte) error {
	if err := IsValidMemo(op.Memo); err != nil {
		return err
	}

	return operation.IsValidOperation(op, networkID)
}

func (op CreateAccounts) GenerateHash() valuehash.Hash {
	bs := make([][]byte, len(op.Signs())+1)
	for i := range op.Signs() {
		bs[i] = op.Signs()[i].Bytes()
	}

	bs[len(bs)-1] = []byte(op.Memo)

	e := util.ConcatBytesSlice(op.Fact().Hash().Bytes(), util.ConcatBytesSlice(bs...))

	return valuehash.NewSHA256(e)
}

func (op CreateAccounts) AddFactSigns(fs ...operation.FactSign) (operation.FactSignUpdater, error) {
	if o, err := op.BaseOperation.AddFactSigns(fs...); err != nil {
		return nil, err
	} else {
		op.BaseOperation = o.(operation.BaseOperation)
	}

	op.BaseOperation = op.SetHash(op.GenerateHash())

	return op, nil
}
