package cmds

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
)

type CurrencyFixedFeeerFlags struct {
	Receiver AddressFlag `name:"receiver" help:"fee receiver account address"`
	Amount   BigFlag     `name:"amount" help:"fee amount"`
	feeer    currency.Feeer
}

func (fl *CurrencyFixedFeeerFlags) IsValid([]byte) error {
	if len(fl.Receiver.String()) < 1 {
		return nil
	}

	var receiver base.Address
	if a, err := fl.Receiver.Encode(jenc); err != nil {
		return errors.Wrapf(err, "invalid receiver format, %q", fl.Receiver.String())
	} else if err := a.IsValid(nil); err != nil {
		return errors.Wrapf(err, "invalid receiver address, %q", fl.Receiver.String())
	} else {
		receiver = a
	}

	fl.feeer = currency.NewFixedFeeer(receiver, fl.Amount.Big)
	return fl.feeer.IsValid(nil)
}

type CurrencyRatioFeeerFlags struct {
	Receiver AddressFlag `name:"receiver" help:"fee receiver account address"`
	Ratio    float64     `name:"ratio" help:"fee ratio, multifly by operation amount"`
	Min      BigFlag     `name:"min" help:"minimum fee"`
	Max      BigFlag     `name:"max" help:"maximum fee"`
	feeer    currency.Feeer
}

func (fl *CurrencyRatioFeeerFlags) IsValid([]byte) error {
	if len(fl.Receiver.String()) < 1 {
		return nil
	}

	var receiver base.Address
	if a, err := fl.Receiver.Encode(jenc); err != nil {
		return errors.Wrapf(err, "invalid receiver format, %q", fl.Receiver.String())
	} else if err := a.IsValid(nil); err != nil {
		return errors.Wrapf(err, "invalid receiver address, %q", fl.Receiver.String())
	} else {
		receiver = a
	}

	fl.feeer = currency.NewRatioFeeer(receiver, fl.Ratio, fl.Min.Big, fl.Max.Big)
	return fl.feeer.IsValid(nil)
}

type CurrencyPolicyFlags struct {
	NewAccountMinBalance BigFlag `name:"new-account-min-balance" help:"minimum balance for new account"` // nolint lll
}

func (*CurrencyPolicyFlags) IsValid([]byte) error {
	return nil
}

type CurrencyDesignFlags struct {
	Currency                CurrencyIDFlag `arg:"" name:"currency-id" help:"currency id" required:"true"`
	GenesisAmount           BigFlag        `arg:"" name:"genesis-amount" help:"genesis amount" required:"true"`
	GenesisAccount          AddressFlag    `arg:"" name:"genesis-account" help:"genesis-account address for genesis balance" required:"true"` // nolint lll
	CurrencyPolicyFlags     `prefix:"policy-" help:"currency policy" required:"true"`
	FeeerString             string `name:"feeer" help:"feeer type, {nil, fixed, ratio}" required:"true"`
	CurrencyFixedFeeerFlags `prefix:"feeer-fixed-" help:"fixed feeer"`
	CurrencyRatioFeeerFlags `prefix:"feeer-ratio-" help:"ratio feeer"`
	currencyDesign          currency.CurrencyDesign
}

func (fl *CurrencyDesignFlags) IsValid([]byte) error {
	if err := fl.CurrencyPolicyFlags.IsValid(nil); err != nil {
		return err
	} else if err := fl.CurrencyFixedFeeerFlags.IsValid(nil); err != nil {
		return err
	} else if err := fl.CurrencyRatioFeeerFlags.IsValid(nil); err != nil {
		return err
	}

	var feeer currency.Feeer
	switch t := fl.FeeerString; t {
	case currency.FeeerNil, "":
		feeer = currency.NewNilFeeer()
	case currency.FeeerFixed:
		feeer = fl.CurrencyFixedFeeerFlags.feeer
	case currency.FeeerRatio:
		feeer = fl.CurrencyRatioFeeerFlags.feeer
	default:
		return errors.Errorf("unknown feeer type, %q", t)
	}

	if feeer == nil {
		return errors.Errorf("empty feeer flags")
	} else if err := feeer.IsValid(nil); err != nil {
		return err
	}

	po := currency.NewCurrencyPolicy(fl.CurrencyPolicyFlags.NewAccountMinBalance.Big, feeer)
	if err := po.IsValid(nil); err != nil {
		return err
	}

	var genesisAccount base.Address
	if a, err := fl.GenesisAccount.Encode(jenc); err != nil {
		return errors.Wrapf(err, "invalid genesis-account format, %q", fl.GenesisAccount.String())
	} else if err := a.IsValid(nil); err != nil {
		return errors.Wrapf(err, "invalid genesis-account address, %q", fl.GenesisAccount.String())
	} else {
		genesisAccount = a
	}

	am := currency.NewAmount(fl.GenesisAmount.Big, fl.Currency.CID)
	if err := am.IsValid(nil); err != nil {
		return err
	}

	fl.currencyDesign = currency.NewCurrencyDesign(am, genesisAccount, po)
	return fl.currencyDesign.IsValid(nil)
}

type CurrencyRegisterCommand struct {
	*BaseCommand
	OperationFlags
	CurrencyDesignFlags
}

func NewCurrencyRegisterCommand() CurrencyRegisterCommand {
	return CurrencyRegisterCommand{
		BaseCommand: NewBaseCommand("currency-register-operation"),
	}
}

func (cmd *CurrencyRegisterCommand) Run(version util.Version) error { // nolint:dupl
	if err := cmd.Initialize(cmd, version); err != nil {
		return errors.Wrap(err, "failed to initialize command")
	}

	if err := cmd.parseFlags(); err != nil {
		return err
	}

	var op operation.Operation
	if i, err := cmd.createOperation(); err != nil {
		return errors.Wrap(err, "failed to create currency-register operation")
	} else if err := i.IsValid([]byte(cmd.OperationFlags.NetworkID)); err != nil {
		return errors.Wrap(err, "invalid currency-register operation")
	} else {
		cmd.Log().Debug().Interface("operation", i).Msg("operation loaded")

		op = i
	}

	i, err := operation.NewBaseSeal(
		cmd.OperationFlags.Privatekey,
		[]operation.Operation{op},
		[]byte(cmd.OperationFlags.NetworkID),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create operation.Seal")
	}
	cmd.Log().Debug().Interface("seal", i).Msg("seal loaded")

	PrettyPrint(cmd.out, cmd.Pretty, i)

	return nil
}

func (cmd *CurrencyRegisterCommand) parseFlags() error {
	if err := cmd.OperationFlags.IsValid(nil); err != nil {
		return err
	} else if err := cmd.CurrencyDesignFlags.IsValid(nil); err != nil {
		return err
	}

	cmd.Log().Debug().Interface("currency-design", cmd.CurrencyDesignFlags.currencyDesign).Msg("currency design loaded")

	return nil
}

func (cmd *CurrencyRegisterCommand) createOperation() (currency.CurrencyRegister, error) {
	fact := currency.NewCurrencyRegisterFact([]byte(cmd.Token), cmd.currencyDesign)

	var fs []operation.FactSign
	sig, err := operation.NewFactSignature(
		cmd.OperationFlags.Privatekey,
		fact,
		[]byte(cmd.OperationFlags.NetworkID),
	)
	if err != nil {
		return currency.CurrencyRegister{}, err
	}
	fs = append(fs, operation.NewBaseFactSign(cmd.OperationFlags.Privatekey.Publickey(), sig))

	return currency.NewCurrencyRegister(fact, fs, cmd.OperationFlags.Memo)
}
