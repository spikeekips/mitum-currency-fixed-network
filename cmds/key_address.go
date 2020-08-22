package cmds

import (
	"github.com/spikeekips/mitum/util/logging"

	"github.com/spikeekips/mitum-currency/currency"
)

type KeyAddressCommand struct {
	printCommand
	Keys      []KeyFlag `arg:"" name:"key" help:"key for address (ex: \"<public key>,<weight>\")" sep:"@" optional:""`
	Threshold uint      `arg:"" name:"threshold" help:"threshold for keys (default: ${create_account_threshold})" default:"${create_account_threshold}"` // nolint
}

func (cmd *KeyAddressCommand) Run(log logging.Logger) error {
	ks := make([]currency.Key, len(cmd.Keys))
	for i := range cmd.Keys {
		ks[i] = cmd.Keys[i].Key
	}

	keys, err := currency.NewKeys(ks, cmd.Threshold)
	if err != nil {
		return err
	}

	log.Debug().Int("number_of_keys", len(ks)).Interface("keys", keys).Msg("keys loaded")

	if a, err := currency.NewAddressFromKeys(keys); err != nil {
		return err
	} else {
		cmd.print(a.String())
	}

	return nil
}
