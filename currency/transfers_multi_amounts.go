package currency

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/hint"
)

var (
	TransfersItemMultiAmountsType   = hint.Type("mitum-currency-transfers-item-multi-amounts")
	TransfersItemMultiAmountsHint   = hint.NewHint(TransfersItemMultiAmountsType, "v0.0.1")
	TransfersItemMultiAmountsHinter = TransfersItemMultiAmounts{
		BaseTransfersItem: BaseTransfersItem{BaseHinter: hint.NewBaseHinter(TransfersItemMultiAmountsHint)},
	}
)

var maxCurenciesTransfersItemMultiAmounts = 10

type TransfersItemMultiAmounts struct {
	BaseTransfersItem
}

func NewTransfersItemMultiAmounts(receiver base.Address, amounts []Amount) TransfersItemMultiAmounts {
	return TransfersItemMultiAmounts{
		BaseTransfersItem: NewBaseTransfersItem(TransfersItemMultiAmountsHint, receiver, amounts),
	}
}

func (it TransfersItemMultiAmounts) IsValid([]byte) error {
	if err := it.BaseTransfersItem.IsValid(nil); err != nil {
		return err
	}

	if n := len(it.amounts); n > maxCurenciesCreateAccountsItemMultiAmounts {
		return errors.Errorf("amounts over allowed; %d > %d", n, maxCurenciesTransfersItemMultiAmounts)
	}

	return nil
}

func (it TransfersItemMultiAmounts) Rebuild() TransfersItem {
	it.BaseTransfersItem = it.BaseTransfersItem.Rebuild().(BaseTransfersItem)

	return it
}
