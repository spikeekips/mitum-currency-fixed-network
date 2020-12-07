package cmds

import (
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum-currency/digest"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/util/hint"
)

var Hinters []hint.Hinter

func init() {
	currencyHinters := []hint.Hinter{
		currency.Address(""),
		currency.Key{},
		currency.Keys{},
		currency.GenesisAccount{},
		currency.GenesisAccountFact{},
		currency.CreateAccounts{},
		currency.CreateAccountsFact{},
		currency.Transfers{},
		currency.TransfersFact{},
		currency.KeyUpdater{},
		currency.KeyUpdaterFact{},
		currency.AmountState{},
		currency.FeeOperationFact{},
		currency.FeeOperation{},
		currency.Account{},
		digest.AccountValue{},
		digest.OperationValue{},
		digest.Problem{},
		digest.BaseHal{},
		digest.NodeInfo{},
	}

	Hinters = make([]hint.Hinter, len(process.DefaultHinters)+len(currencyHinters))
	copy(Hinters, process.DefaultHinters)
	copy(Hinters[len(process.DefaultHinters):], currencyHinters)
}
