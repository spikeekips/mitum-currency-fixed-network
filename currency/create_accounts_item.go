package currency

import (
	"github.com/pkg/errors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
)

type BaseCreateAccountsItem struct {
	hint    hint.Hint
	keys    Keys
	amounts []Amount
}

func NewBaseCreateAccountsItem(ht hint.Hint, keys Keys, amounts []Amount) BaseCreateAccountsItem {
	return BaseCreateAccountsItem{
		hint:    ht,
		keys:    keys,
		amounts: amounts,
	}
}

func (it BaseCreateAccountsItem) Hint() hint.Hint {
	return it.hint
}

func (it BaseCreateAccountsItem) Bytes() []byte {
	bs := make([][]byte, len(it.amounts)+1)
	bs[0] = it.keys.Bytes()

	for i := range it.amounts {
		bs[i+1] = it.amounts[i].Bytes()
	}

	return util.ConcatBytesSlice(bs...)
}

func (it BaseCreateAccountsItem) IsValid([]byte) error {
	if err := it.keys.IsValid(nil); err != nil {
		return err
	}

	if n := len(it.amounts); n == 0 {
		return errors.Errorf("empty amounts")
	}

	founds := map[CurrencyID]struct{}{}
	for i := range it.amounts {
		am := it.amounts[i]
		if _, found := founds[am.Currency()]; found {
			return errors.Errorf("duplicated currency found, %q", am.Currency())
		}
		founds[am.Currency()] = struct{}{}

		if err := am.IsValid(nil); err != nil {
			return err
		} else if !am.Big().OverZero() {
			return errors.Errorf("amount should be over zero")
		}
	}

	return nil
}

func (it BaseCreateAccountsItem) Keys() Keys {
	return it.keys
}

func (it BaseCreateAccountsItem) Address() (base.Address, error) {
	return NewAddressFromKeys(it.keys)
}

func (it BaseCreateAccountsItem) Amounts() []Amount {
	return it.amounts
}

func (it BaseCreateAccountsItem) Rebuild() CreateAccountsItem {
	ams := make([]Amount, len(it.amounts))
	for i := range it.amounts {
		am := it.amounts[i]
		ams[i] = am.WithBig(am.Big())
	}

	it.amounts = ams

	return it
}
