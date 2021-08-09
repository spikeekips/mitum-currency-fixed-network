package cmds

import (
	"github.com/pkg/errors"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
)

type CurrencyPolicyUpdaterCommand struct {
	*BaseCommand
	OperationFlags
	Currency                CurrencyIDFlag `arg:"" name:"currency-id" help:"currency id" required:"true"`
	CurrencyPolicyFlags     `prefix:"policy-" help:"currency policy" required:"true"`
	FeeerString             string `name:"feeer" help:"feeer type, {nil, fixed, ratio}" required:"true"`
	CurrencyFixedFeeerFlags `prefix:"feeer-fixed-" help:"fixed feeer"`
	CurrencyRatioFeeerFlags `prefix:"feeer-ratio-" help:"ratio feeer"`
	po                      currency.CurrencyPolicy
}

func NewCurrencyPolicyUpdaterCommand() CurrencyPolicyUpdaterCommand {
	return CurrencyPolicyUpdaterCommand{
		BaseCommand: NewBaseCommand("currency-policy-updater-operation"),
	}
}

func (cmd *CurrencyPolicyUpdaterCommand) Run(version util.Version) error { // nolint:dupl
	if err := cmd.Initialize(cmd, version); err != nil {
		return errors.Wrap(err, "failed to initialize command")
	}

	if err := cmd.parseFlags(); err != nil {
		return err
	}

	var op operation.Operation
	if i, err := cmd.createOperation(); err != nil {
		return errors.Wrap(err, "failed to create currency-policy-updater operation")
	} else if err := i.IsValid([]byte(cmd.OperationFlags.NetworkID)); err != nil {
		return errors.Wrap(err, "invalid currency-policy-updater operation")
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

	cmd.pretty(cmd.Pretty, i)

	return nil
}

func (cmd *CurrencyPolicyUpdaterCommand) parseFlags() error {
	if err := cmd.OperationFlags.IsValid(nil); err != nil {
		return err
	} else if err := cmd.CurrencyPolicyFlags.IsValid(nil); err != nil {
		return err
	}

	if err := cmd.CurrencyFixedFeeerFlags.IsValid(nil); err != nil {
		return err
	} else if err := cmd.CurrencyRatioFeeerFlags.IsValid(nil); err != nil {
		return err
	}

	var feeer currency.Feeer
	switch t := cmd.FeeerString; t {
	case currency.FeeerNil, "":
		feeer = currency.NewNilFeeer()
	case currency.FeeerFixed:
		feeer = cmd.CurrencyFixedFeeerFlags.feeer
	case currency.FeeerRatio:
		feeer = cmd.CurrencyRatioFeeerFlags.feeer
	default:
		return errors.Errorf("unknown feeer type, %q", t)
	}

	if feeer == nil {
		return errors.Errorf("empty feeer flags")
	} else if err := feeer.IsValid(nil); err != nil {
		return err
	}

	cmd.po = currency.NewCurrencyPolicy(cmd.CurrencyPolicyFlags.NewAccountMinBalance.Big, feeer)
	if err := cmd.po.IsValid(nil); err != nil {
		return err
	}

	cmd.Log().Debug().Interface("currency-policy", cmd.po).Msg("currency policy loaded")

	return nil
}

func (cmd *CurrencyPolicyUpdaterCommand) createOperation() (currency.CurrencyPolicyUpdater, error) {
	fact := currency.NewCurrencyPolicyUpdaterFact([]byte(cmd.Token), cmd.Currency.CID, cmd.po)

	var fs []operation.FactSign
	sig, err := operation.NewFactSignature(
		cmd.OperationFlags.Privatekey,
		fact,
		[]byte(cmd.OperationFlags.NetworkID),
	)
	if err != nil {
		return currency.CurrencyPolicyUpdater{}, err
	}
	fs = append(fs, operation.NewBaseFactSign(cmd.OperationFlags.Privatekey.Publickey(), sig))

	return currency.NewCurrencyPolicyUpdater(fact, fs, cmd.OperationFlags.Memo)
}
