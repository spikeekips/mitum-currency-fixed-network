package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	TransfersItemSingleAmountType   = hint.Type("mitum-currency-transfers-item-single-amount")
	TransfersItemSingleAmountHint   = hint.NewHint(TransfersItemSingleAmountType, "v0.0.1")
	TransfersItemSingleAmountHinter = TransfersItemSingleAmount{
		BaseTransfersItem: BaseTransfersItem{BaseHinter: hint.NewBaseHinter(TransfersItemSingleAmountHint)},
	}
)

type TransfersItemSingleAmount struct {
	BaseTransfersItem
}

func NewTransfersItemSingleAmount(receiver base.Address, amount Amount) TransfersItemSingleAmount {
	return TransfersItemSingleAmount{
		BaseTransfersItem: NewBaseTransfersItem(TransfersItemSingleAmountHint, receiver, []Amount{amount}),
	}
}

func (it TransfersItemSingleAmount) IsValid([]byte) error {
	if err := it.BaseTransfersItem.IsValid(nil); err != nil {
		return err
	}

	if n := len(it.amounts); n != 1 {
		return isvalid.InvalidError.Errorf("only one amount allowed; %d", n)
	}

	return nil
}

func (it TransfersItemSingleAmount) Rebuild() TransfersItem {
	it.BaseTransfersItem = it.BaseTransfersItem.Rebuild().(BaseTransfersItem)

	return it
}
