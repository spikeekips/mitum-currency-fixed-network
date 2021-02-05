package cmds

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
)

type CurrencyPolicyUpdaterCommand struct {
	*BaseCommand
	OperationFlags
	Currency                CurrencyIDFlag `arg:"" name:"currency-id" help:"currency id" required:""`
	CurrencyPolicyFlags     `prefix:"policy-" help:"currency policy" required:""`
	FeeerString             string `name:"feeer" help:"feeer type, {nil, fixed, ratio}" required:""`
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
		return xerrors.Errorf("failed to initialize command: %w", err)
	}

	if err := cmd.parseFlags(); err != nil {
		return err
	}

	var op operation.Operation
	if i, err := cmd.createOperation(); err != nil {
		return xerrors.Errorf("failed to create currency-policy-updater operation: %w", err)
	} else if err := i.IsValid([]byte(cmd.OperationFlags.NetworkID)); err != nil {
		return xerrors.Errorf("invalid currency-policy-updater operation: %w", err)
	} else {
		cmd.Log().Debug().Interface("operation", i).Msg("operation loaded")

		op = i
	}

	if i, err := operation.NewBaseSeal(
		cmd.OperationFlags.Privatekey,
		[]operation.Operation{op},
		[]byte(cmd.OperationFlags.NetworkID),
	); err != nil {
		return xerrors.Errorf("failed to create operation.Seal: %w", err)
	} else {
		cmd.Log().Debug().Interface("seal", i).Msg("seal loaded")

		cmd.pretty(cmd.Pretty, i)
	}

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
		return xerrors.Errorf("unknown feeer type, %q", t)
	}

	if feeer == nil {
		return xerrors.Errorf("empty feeer flags")
	} else if err := feeer.IsValid(nil); err != nil {
		return err
	}

	po := currency.NewCurrencyPolicy(cmd.CurrencyPolicyFlags.NewAccountMinBalance.Big, feeer)
	if err := po.IsValid(nil); err != nil {
		return err
	} else {
		cmd.po = po
	}

	cmd.Log().Debug().Interface("currency-policy", cmd.po).Msg("currency policy loaded")

	return nil
}

func (cmd *CurrencyPolicyUpdaterCommand) createOperation() (currency.CurrencyPolicyUpdater, error) {
	fact := currency.NewCurrencyPolicyUpdaterFact([]byte(cmd.Token), cmd.Currency.CID, cmd.po)

	var fs []operation.FactSign
	if sig, err := operation.NewFactSignature(
		cmd.OperationFlags.Privatekey,
		fact,
		[]byte(cmd.OperationFlags.NetworkID),
	); err != nil {
		return currency.CurrencyPolicyUpdater{}, err
	} else {
		fs = append(fs, operation.NewBaseFactSign(cmd.OperationFlags.Privatekey.Publickey(), sig))
	}

	return currency.NewCurrencyPolicyUpdater(fact, fs, cmd.OperationFlags.Memo)
}
