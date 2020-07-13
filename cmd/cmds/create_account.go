package cmds

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util/localtime"

	mc "github.com/spikeekips/mitum-currency"
)

type CreateAccountCommand struct {
	Privatekey PrivatekeyFlag `arg:"" name:"privatekey" help:"sender's privatekey" required:""`
	Sender     AddressFlag    `arg:"" name:"sender" help:"sender address" required:""`
	Amount     AmountFlag     `arg:"" name:"amount" help:"amount to send" required:""`
	Threshold  uint           `help:"threshold for keys (default: ${create_account_threshold})" default:"${create_account_threshold}"` // nolint
	Token      string         `help:"token for operation" optional:""`
	NetworkID  string         `name:"network-id" help:"network-id" required:""`
	Keys       []KeyFlag      `name:"key" help:"key for new account (ex: \"<public key>,<weight>\")" sep:"@"`
	Pretty     bool           `name:"pretty" help:"pretty format"`
	Memo       string         `name:"memo" help:"memo"`

	keys mc.Keys
}

func (cmd *CreateAccountCommand) Run() error {
	if err := cmd.parseFlags(); err != nil {
		return err
	}

	var sl operation.Seal
	if s, err := cmd.createSeal(); err != nil {
		return err
	} else {
		sl = s
	}

	prettyPrint(cmd.Pretty, sl)

	return nil
}

func (cmd *CreateAccountCommand) parseFlags() error {
	if len(cmd.Keys) < 1 {
		return xerrors.Errorf("--key must be given at least one")
	}

	if len(cmd.Token) < 1 {
		cmd.Token = localtime.String(localtime.Now())
	}

	{
		ks := make([]mc.Key, len(cmd.Keys))
		for i := range cmd.Keys {
			ks[i] = cmd.Keys[i].Key
		}

		if kys, err := mc.NewKeys(ks, cmd.Threshold); err != nil {
			return err
		} else if err := kys.IsValid(nil); err != nil {
			return err
		} else {
			cmd.keys = kys
		}
	}

	if len(cmd.Memo) < 1 {
		if b, err := loadFromStdInput(); err != nil {
			return err
		} else {
			cmd.Memo = string(b)
		}
	}

	return nil
}

func (cmd *CreateAccountCommand) createSeal() (operation.Seal, error) {
	fact := mc.NewCreateAccountFact([]byte(cmd.Token), cmd.Sender.Address, cmd.keys, cmd.Amount.Amount)

	var fs []operation.FactSign
	if sig, err := operation.NewFactSignature(cmd.Privatekey, fact, []byte(cmd.NetworkID)); err != nil {
		return nil, err
	} else {
		fs = append(fs, operation.NewBaseFactSign(cmd.Privatekey.Publickey(), sig))
	}

	if op, err := mc.NewCreateAccount(fact, fs, cmd.Memo); err != nil {
		return nil, xerrors.Errorf("failed to create create-account operation: %w", err)
	} else if sl, err := operation.NewBaseSeal(
		cmd.Privatekey,
		[]operation.Operation{op},
		[]byte(cmd.NetworkID),
	); err != nil {
		return nil, xerrors.Errorf("failed to create operation.Seal: %w", err)
	} else {
		return sl, nil
	}
}
