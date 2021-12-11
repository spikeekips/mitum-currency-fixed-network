package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

type BaseTransfersItem struct {
	hint.BaseHinter
	receiver base.Address
	amounts  []Amount
}

func NewBaseTransfersItem(ht hint.Hint, receiver base.Address, amounts []Amount) BaseTransfersItem {
	return BaseTransfersItem{
		BaseHinter: hint.NewBaseHinter(ht),
		receiver:   receiver,
		amounts:    amounts,
	}
}

func (it BaseTransfersItem) Bytes() []byte {
	bs := make([][]byte, len(it.amounts)+1)
	bs[0] = it.receiver.Bytes()

	for i := range it.amounts {
		bs[i+1] = it.amounts[i].Bytes()
	}

	return util.ConcatBytesSlice(bs...)
}

func (it BaseTransfersItem) IsValid([]byte) error {
	if err := isvalid.Check(nil, false, it.receiver); err != nil {
		return err
	}

	if n := len(it.amounts); n == 0 {
		return isvalid.InvalidError.Errorf("empty amounts")
	}

	founds := map[CurrencyID]struct{}{}
	for i := range it.amounts {
		am := it.amounts[i]
		if _, found := founds[am.Currency()]; found {
			return isvalid.InvalidError.Errorf("duplicated currency found, %q", am.Currency())
		}
		founds[am.Currency()] = struct{}{}

		if err := am.IsValid(nil); err != nil {
			return err
		} else if !am.Big().OverZero() {
			return isvalid.InvalidError.Errorf("amount should be over zero")
		}
	}

	return nil
}

func (it BaseTransfersItem) Receiver() base.Address {
	return it.receiver
}

func (it BaseTransfersItem) Amounts() []Amount {
	return it.amounts
}

func (it BaseTransfersItem) Rebuild() TransfersItem {
	ams := make([]Amount, len(it.amounts))
	for i := range it.amounts {
		am := it.amounts[i]
		ams[i] = am.WithBig(am.Big())
	}

	it.amounts = ams

	return it
}
