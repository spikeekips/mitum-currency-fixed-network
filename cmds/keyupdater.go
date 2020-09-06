package cmds

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util/localtime"

	"github.com/spikeekips/mitum-currency/currency"
)

type KeyUpdaterCommand struct {
	printCommand
	Privatekey PrivatekeyFlag `arg:"" name:"privatekey" help:"target's privatekey" required:""`
	Target     AddressFlag    `arg:"" name:"target" help:"target address" required:""`
	Threshold  uint           `help:"threshold for keys (default: ${create_account_threshold})" default:"${create_account_threshold}"` // nolint
	Token      string         `help:"token for operation" optional:""`
	NetworkID  NetworkIDFlag  `name:"network-id" help:"network-id" required:""`
	Keys       []KeyFlag      `name:"key" help:"key for account (ex: \"<public key>,<weight>\")" sep:"@"`
	Pretty     bool           `name:"pretty" help:"pretty format"`
	Memo       string         `name:"memo" help:"memo"`
	Seal       FileLoad       `help:"seal" optional:""`

	keys currency.Keys
}

func (cmd *KeyUpdaterCommand) Run() error {
	if err := cmd.parseFlags(); err != nil {
		return err
	}

	var op operation.Operation
	if o, err := cmd.createOperation(); err != nil {
		return err
	} else {
		op = o
	}

	if sl, err := loadSealAndAddOperation(
		cmd.Seal.Bytes(),
		cmd.Privatekey,
		cmd.NetworkID.Bytes(),
		op,
	); err != nil {
		return err
	} else {
		cmd.pretty(cmd.Pretty, sl)
	}

	return nil
}

func (cmd *KeyUpdaterCommand) parseFlags() error {
	if len(cmd.Keys) < 1 {
		return xerrors.Errorf("--key must be given at least one")
	}

	if len(cmd.Token) < 1 {
		cmd.Token = localtime.String(localtime.Now())
	}

	{
		ks := make([]currency.Key, len(cmd.Keys))
		for i := range cmd.Keys {
			ks[i] = cmd.Keys[i].Key
		}

		if kys, err := currency.NewKeys(ks, cmd.Threshold); err != nil {
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
		cmd.Target.Address,
		cmd.keys,
	)

	var fs []operation.FactSign
	if sig, err := operation.NewFactSignature(cmd.Privatekey, fact, []byte(cmd.NetworkID)); err != nil {
		return nil, err
	} else {
		fs = append(fs, operation.NewBaseFactSign(cmd.Privatekey.Publickey(), sig))
	}

	if op, err := currency.NewKeyUpdater(fact, fs, cmd.Memo); err != nil {
		return nil, xerrors.Errorf("failed to create key-updater operation: %w", err)
	} else {
		return op, nil
	}
}
