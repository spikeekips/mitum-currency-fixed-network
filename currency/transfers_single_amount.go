package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/hint"
	"golang.org/x/xerrors"
)

var (
	TransfersItemSingleAmountType   = hint.MustNewType(0xa0, 0x27, "mitum-currency-transfers-item-single-amount")
	TransfersItemSingleAmountHint   = hint.MustHint(TransfersItemSingleAmountType, "0.0.1")
	TransfersItemSingleAmountHinter = BaseTransfersItem{hint: TransfersItemSingleAmountHint}
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
		return xerrors.Errorf("only one amount allowed; %d", n)
	}

	return nil
}

func (it TransfersItemSingleAmount) Rebuild() TransfersItem {
	it.BaseTransfersItem = it.BaseTransfersItem.Rebuild().(BaseTransfersItem)

	return it
}
