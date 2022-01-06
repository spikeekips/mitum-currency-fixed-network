package currency

import (
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

var maxCurenciesCreateAccountsItemMultiAmounts = 10

var (
	CreateAccountsItemMultiAmountsType   = hint.Type("mitum-currency-create-accounts-multiple-amounts")
	CreateAccountsItemMultiAmountsHint   = hint.NewHint(CreateAccountsItemMultiAmountsType, "v0.0.1")
	CreateAccountsItemMultiAmountsHinter = CreateAccountsItemMultiAmounts{
		BaseCreateAccountsItem: BaseCreateAccountsItem{
			BaseHinter: hint.NewBaseHinter(CreateAccountsItemMultiAmountsHint),
		},
	}
)

type CreateAccountsItemMultiAmounts struct {
	BaseCreateAccountsItem
}

func NewCreateAccountsItemMultiAmounts(keys AccountKeys, amounts []Amount) CreateAccountsItemMultiAmounts {
	return CreateAccountsItemMultiAmounts{
		BaseCreateAccountsItem: NewBaseCreateAccountsItem(CreateAccountsItemMultiAmountsHint, keys, amounts),
	}
}

func (it CreateAccountsItemMultiAmounts) IsValid([]byte) error {
	if err := it.BaseCreateAccountsItem.IsValid(nil); err != nil {
		return err
	}

	if n := len(it.amounts); n > maxCurenciesCreateAccountsItemMultiAmounts {
		return isvalid.InvalidError.Errorf("amounts over allowed; %d > %d", n, maxCurenciesCreateAccountsItemMultiAmounts)
	}

	return nil
}

func (it CreateAccountsItemMultiAmounts) Rebuild() CreateAccountsItem {
	it.BaseCreateAccountsItem = it.BaseCreateAccountsItem.Rebuild().(BaseCreateAccountsItem)

	return it
}
