package cmds

import (
	"github.com/pkg/errors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"

	"github.com/spikeekips/mitum-currency/currency"
)

type KeyUpdaterCommand struct {
	*BaseCommand
	OperationFlags
	Target    AddressFlag    `arg:"" name:"target" help:"target address" required:"true"`
	Currency  CurrencyIDFlag `arg:"" name:"currency" help:"currency id" required:"true"`
	Threshold uint           `help:"threshold for keys (default: ${create_account_threshold})" default:"${create_account_threshold}"` // nolint
	Keys      []KeyFlag      `name:"key" help:"key for account (ex: \"<public key>,<weight>\")" sep:"@"`
	target    base.Address
	keys      currency.BaseAccountKeys
}

func NewKeyUpdaterCommand() KeyUpdaterCommand {
	return KeyUpdaterCommand{
		BaseCommand: NewBaseCommand("keyupdater-operation"),
	}
}

func (cmd *KeyUpdaterCommand) Run(version util.Version) error { // nolint:dupl
	if err := cmd.Initialize(cmd, version); err != nil {
		return errors.Wrap(err, "failed to initialize command")
	}

	if err := cmd.parseFlags(); err != nil {
		return err
	}

	op, err := cmd.createOperation()
	if err != nil {
		return err
	}

	bs, err := operation.NewBaseSeal(
		cmd.Privatekey,
		[]operation.Operation{op},
		cmd.NetworkID.NetworkID(),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create operation.Seal")
	}
	PrettyPrint(cmd.Out, cmd.Pretty, bs)

	return nil
}

func (cmd *KeyUpdaterCommand) parseFlags() error {
	if err := cmd.OperationFlags.IsValid(nil); err != nil {
		return err
	}

	if len(cmd.Keys) < 1 {
		return errors.Errorf("--key must be given at least one")
	}

	a, err := cmd.Target.Encode(jenc)
	if err != nil {
		return errors.Wrapf(err, "invalid target format, %q", cmd.Target.String())
	}
	cmd.target = a

	{
		ks := make([]currency.AccountKey, len(cmd.Keys))
		for i := range cmd.Keys {
			ks[i] = cmd.Keys[i].Key
		}

		if kys, err := currency.NewBaseAccountKeys(ks, cmd.Threshold); err != nil {
			return err
		} else if err := kys.IsValid(nil); err != nil {
			return err
		} else {
			cmd.keys = kys
		}
	}

	return nil
}

func (cmd *KeyUpdaterCommand) createOperation() (operation.Operation, error) {
	fact := currency.NewKeyUpdaterFact(
		[]byte(cmd.Token),
		cmd.target,
		cmd.keys,
		cmd.Currency.CID,
	)

	var fs []base.FactSign
	sig, err := base.NewFactSignature(cmd.Privatekey, fact, cmd.NetworkID.NetworkID())
	if err != nil {
		return nil, err
	}
	fs = append(fs, base.NewBaseFactSign(cmd.Privatekey.Publickey(), sig))

	op, err := currency.NewKeyUpdater(fact, fs, cmd.Memo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create key-updater operation")
	}
	return op, nil
}
