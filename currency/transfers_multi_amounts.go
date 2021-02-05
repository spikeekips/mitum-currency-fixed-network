package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/hint"
	"golang.org/x/xerrors"
)

var (
	TransfersItemMultiAmountsType   = hint.MustNewType(0xa0, 0x26, "mitum-currency-transfers-item-multi-amounts")
	TransfersItemMultiAmountsHint   = hint.MustHint(TransfersItemMultiAmountsType, "0.0.1")
	TransfersItemMultiAmountsHinter = BaseTransfersItem{hint: TransfersItemMultiAmountsHint}
)

var maxCurenciesTransfersItemMultiAmounts int = 10

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
		return xerrors.Errorf("amounts over allowed; %d > %d", n, maxCurenciesTransfersItemMultiAmounts)
	}

	return nil
}

func (it TransfersItemMultiAmounts) Rebuild() TransfersItem {
	it.BaseTransfersItem = it.BaseTransfersItem.Rebuild().(BaseTransfersItem)

	return it
}
