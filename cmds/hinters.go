package cmds

import (
	"github.com/spikeekips/mitum/launch"
	"github.com/spikeekips/mitum/util/hint"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum-currency/digest"
)

var (
	Hinters []hint.Hinter
	Types   []hint.Type
)

var types = []hint.Type{
	currency.AccountType,
	currency.AddressType,
	currency.AmountStateType,
	currency.AmountType,
	currency.CreateAccountsFactType,
	currency.CreateAccountsItemMultiAmountsType,
	currency.CreateAccountsItemSingleAmountType,
	currency.CreateAccountsType,
	currency.CurrencyDesignType,
	currency.CurrencyPolicyType,
	currency.CurrencyPolicyUpdaterFactType,
	currency.CurrencyPolicyUpdaterType,
	currency.CurrencyRegisterFactType,
	currency.CurrencyRegisterType,
	currency.FeeOperationFactType,
	currency.FeeOperationType,
	currency.FixedFeeerType,
	currency.GenesisCurrenciesFactType,
	currency.GenesisCurrenciesType,
	currency.KeyType,
	currency.KeyUpdaterFactType,
	currency.KeyUpdaterType,
	currency.KeysType,
	currency.NilFeeerType,
	currency.RatioFeeerType,
	currency.SuffrageInflationFactType,
	currency.SuffrageInflationType,
	currency.TransfersFactType,
	currency.TransfersItemMultiAmountsType,
	currency.TransfersItemSingleAmountType,
	currency.TransfersType,
	digest.ProblemType,
	digest.NodeInfoType,
	digest.BaseHalType,
	digest.AccountValueType,
	digest.OperationValueType,
}

var hinters = []hint.Hinter{
	currency.Account{},
	currency.Address(""),
	currency.AmountState{},
	currency.Amount{},
	currency.CreateAccountsFact{},
	currency.CreateAccountsItemMultiAmountsHinter,
	currency.CreateAccountsItemSingleAmountHinter,
	currency.CreateAccountsHinter,
	currency.CurrencyDesign{},
	currency.CurrencyPolicyUpdaterFact{},
	currency.CurrencyPolicyUpdaterHinter,
	currency.CurrencyPolicy{},
	currency.CurrencyRegisterFact{},
	currency.CurrencyRegisterHinter,
	currency.FeeOperationFact{},
	currency.FeeOperation{},
	currency.FixedFeeer{},
	currency.GenesisCurrenciesFact{},
	currency.GenesisCurrenciesHinter,
	currency.KeyUpdaterFact{},
	currency.KeyUpdaterHinter,
	currency.Keys{},
	currency.Key{},
	currency.NilFeeer{},
	currency.RatioFeeer{},
	currency.SuffrageInflationFact{},
	currency.SuffrageInflationHinter,
	currency.TransfersFact{},
	currency.TransfersItemMultiAmountsHinter,
	currency.TransfersItemSingleAmountHinter,
	currency.TransfersHinter,
	digest.AccountValue{},
	digest.BaseHal{},
	digest.NodeInfo{},
	digest.OperationValue{},
	digest.Problem{},
}

func init() {
	Hinters = make([]hint.Hinter, len(launch.EncoderHinters)+len(hinters))
	copy(Hinters, launch.EncoderHinters)
	copy(Hinters[len(launch.EncoderHinters):], hinters)

	Types = make([]hint.Type, len(launch.EncoderTypes)+len(types))
	copy(Types, launch.EncoderTypes)
	copy(Types[len(launch.EncoderTypes):], types)
}
