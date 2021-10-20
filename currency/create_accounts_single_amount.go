package currency

import (
	"github.com/pkg/errors"

	"github.com/spikeekips/mitum/util/hint"
)

var (
	CreateAccountsItemSingleAmountType   = hint.Type("mitum-currency-create-accounts-single-amount")
	CreateAccountsItemSingleAmountHint   = hint.NewHint(CreateAccountsItemSingleAmountType, "v0.0.1")
	CreateAccountsItemSingleAmountHinter = CreateAccountsItemSingleAmount{
		BaseCreateAccountsItem: BaseCreateAccountsItem{hint: CreateAccountsItemSingleAmountHint},
	}
)

type CreateAccountsItemSingleAmount struct {
	BaseCreateAccountsItem
}

func NewCreateAccountsItemSingleAmount(keys Keys, amount Amount) CreateAccountsItemSingleAmount {
	return CreateAccountsItemSingleAmount{
		BaseCreateAccountsItem: NewBaseCreateAccountsItem(CreateAccountsItemSingleAmountHint, keys, []Amount{amount}),
	}
}

func (it CreateAccountsItemSingleAmount) IsValid([]byte) error {
	if err := it.BaseCreateAccountsItem.IsValid(nil); err != nil {
		return err
	}

	if n := len(it.amounts); n != 1 {
		return errors.Errorf("only one amount allowed; %d", n)
	}

	return nil
}

func (it CreateAccountsItemSingleAmount) Rebuild() CreateAccountsItem {
	it.BaseCreateAccountsItem = it.BaseCreateAccountsItem.Rebuild().(BaseCreateAccountsItem)

	return it
}
