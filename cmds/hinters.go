package cmds

import (
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/util/hint"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum-currency/digest"
)

var Hinters []hint.Hinter

func init() {
	currencyHinters := []hint.Hinter{
		currency.Account{},
		currency.Address(""),
		currency.AmountState{},
		currency.Amount{},
		currency.CreateAccountsFact{},
		currency.CreateAccountsItemMultiAmountsHinter,
		currency.CreateAccountsItemSingleAmountHinter,
		currency.CreateAccounts{},
		currency.CurrencyDesign{},
		currency.CurrencyPolicyUpdaterFact{},
		currency.CurrencyPolicyUpdater{},
		currency.CurrencyPolicy{},
		currency.CurrencyRegisterFact{},
		currency.CurrencyRegister{},
		currency.FeeOperationFact{},
		currency.FeeOperation{},
		currency.FixedFeeer{},
		currency.GenesisCurrenciesFact{},
		currency.GenesisCurrencies{},
		currency.KeyUpdaterFact{},
		currency.KeyUpdater{},
		currency.Keys{},
		currency.Key{},
		currency.NilFeeer{},
		currency.RatioFeeer{},
		currency.TransfersFact{},
		currency.TransfersItemMultiAmountsHinter,
		currency.TransfersItemSingleAmountHinter,
		currency.Transfers{},
		digest.AccountValue{},
		digest.BaseHal{},
		digest.NodeInfo{},
		digest.OperationValue{},
		digest.Problem{},
	}

	Hinters = make([]hint.Hinter, len(process.DefaultHinters)+len(currencyHinters))
	copy(Hinters, process.DefaultHinters)
	copy(Hinters[len(process.DefaultHinters):], currencyHinters)
}
