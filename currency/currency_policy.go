package currency

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"golang.org/x/xerrors"
)

var (
	CurrencyPolicyType = hint.Type("mitum-currency-currency-policy")
	CurrencyPolicyHint = hint.NewHint(CurrencyPolicyType, "v0.0.1")
)

type CurrencyPolicy struct {
	newAccountMinBalance Big
	feeer                Feeer
}

func NewCurrencyPolicy(newAccountMinBalance Big, feeer Feeer) CurrencyPolicy {
	return CurrencyPolicy{newAccountMinBalance: newAccountMinBalance, feeer: feeer}
}

func (CurrencyPolicy) Hint() hint.Hint {
	return CurrencyPolicyHint
}

func (po CurrencyPolicy) Bytes() []byte {
	return util.ConcatBytesSlice(po.newAccountMinBalance.Bytes(), po.feeer.Bytes())
}

func (po CurrencyPolicy) IsValid([]byte) error {
	if !po.newAccountMinBalance.OverNil() {
		return xerrors.Errorf("NewAccountMinBalance under zero")
	}

	return po.feeer.IsValid(nil)
}

func (po CurrencyPolicy) NewAccountMinBalance() Big {
	return po.newAccountMinBalance
}

func (po CurrencyPolicy) Feeer() Feeer {
	return po.feeer
}
