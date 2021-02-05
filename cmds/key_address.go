package cmds

import (
	"github.com/alecthomas/kong"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"

	"github.com/spikeekips/mitum-currency/currency"
)

var KeyAddressVars = kong.Vars{
	"create_account_threshold": "100",
}

type KeyAddressCommand struct {
	*BaseCommand
	Threshold uint      `arg:"" name:"threshold" help:"threshold for keys (default: ${create_account_threshold})" default:"${create_account_threshold}"` // nolint
	Keys      []KeyFlag `arg:"" name:"key" help:"key for address (ex: \"<public key>,<weight>\")" sep:"@" optional:""`
}

func NewKeyAddressCommand() KeyAddressCommand {
	return KeyAddressCommand{
		BaseCommand: NewBaseCommand("key-address"),
	}
}

func (cmd *KeyAddressCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return xerrors.Errorf("failed to initialize command: %w", err)
	}

	ks := make([]currency.Key, len(cmd.Keys))
	for i := range cmd.Keys {
		ks[i] = cmd.Keys[i].Key
	}

	keys, err := currency.NewKeys(ks, cmd.Threshold)
	if err != nil {
		return err
	}

	cmd.Log().Debug().Int("number_of_keys", len(ks)).Interface("keys", keys).Msg("keys loaded")

	if a, err := currency.NewAddressFromKeys(keys); err != nil {
		return err
	} else {
		cmd.print(a.String())
	}

	return nil
}
