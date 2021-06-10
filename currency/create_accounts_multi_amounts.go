package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/hint"
)

var maxCurenciesCreateAccountsItemMultiAmounts = 10

var (
	CreateAccountsItemMultiAmountsType   = hint.Type("mitum-currency-create-accounts-multiple-amounts")
	CreateAccountsItemMultiAmountsHint   = hint.NewHint(CreateAccountsItemMultiAmountsType, "v0.0.1")
	CreateAccountsItemMultiAmountsHinter = BaseCreateAccountsItem{hint: CreateAccountsItemMultiAmountsHint}
)

type CreateAccountsItemMultiAmounts struct {
	BaseCreateAccountsItem
}

func NewCreateAccountsItemMultiAmounts(keys Keys, amounts []Amount) CreateAccountsItemMultiAmounts {
	return CreateAccountsItemMultiAmounts{
		BaseCreateAccountsItem: NewBaseCreateAccountsItem(CreateAccountsItemMultiAmountsHint, keys, amounts),
	}
}

func (it CreateAccountsItemMultiAmounts) IsValid([]byte) error {
	if err := it.BaseCreateAccountsItem.IsValid(nil); err != nil {
		return err
	}

	if n := len(it.amounts); n > maxCurenciesCreateAccountsItemMultiAmounts {
		return xerrors.Errorf("amounts over allowed; %d > %d", n, maxCurenciesCreateAccountsItemMultiAmounts)
	}

	return nil
}

func (it CreateAccountsItemMultiAmounts) Rebuild() CreateAccountsItem {
	it.BaseCreateAccountsItem = it.BaseCreateAccountsItem.Rebuild().(BaseCreateAccountsItem)

	return it
}
