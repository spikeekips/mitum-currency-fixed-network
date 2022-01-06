package currency

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	CurrencyPolicyType   = hint.Type("mitum-currency-currency-policy")
	CurrencyPolicyHint   = hint.NewHint(CurrencyPolicyType, "v0.0.1")
	CurrencyPolicyHinter = CurrencyPolicy{BaseHinter: hint.NewBaseHinter(CurrencyPolicyHint)}
)

type CurrencyPolicy struct {
	hint.BaseHinter
	newAccountMinBalance Big
	feeer                Feeer
}

func NewCurrencyPolicy(newAccountMinBalance Big, feeer Feeer) CurrencyPolicy {
	return CurrencyPolicy{
		BaseHinter:           hint.NewBaseHinter(CurrencyPolicyHint),
		newAccountMinBalance: newAccountMinBalance, feeer: feeer,
	}
}

func (po CurrencyPolicy) Bytes() []byte {
	return util.ConcatBytesSlice(po.newAccountMinBalance.Bytes(), po.feeer.Bytes())
}

func (po CurrencyPolicy) IsValid([]byte) error {
	if !po.newAccountMinBalance.OverNil() {
		return isvalid.InvalidError.Errorf("NewAccountMinBalance under zero")
	}

	if err := isvalid.Check(nil, false, po.BaseHinter, po.feeer); err != nil {
		return isvalid.InvalidError.Errorf("invalid currency policy: %w", err)
	}

	return nil
}

func (po CurrencyPolicy) NewAccountMinBalance() Big {
	return po.newAccountMinBalance
}

func (po CurrencyPolicy) Feeer() Feeer {
	return po.feeer
}
