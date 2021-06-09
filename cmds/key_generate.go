package cmds

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
)

type GenerateKeyCommand struct {
	*BaseCommand
	Type   string `name:"type" help:"key type {btc ether stellar} (default: btc)" optional:"" default:"btc"`
	JSON   bool   `name:"json" help:"json output format (default: false)" optional:"" default:"false"`
	Pretty bool   `name:"pretty" help:"pretty format"`
}

func NewGenerateKeyCommand() GenerateKeyCommand {
	return GenerateKeyCommand{
		BaseCommand: NewBaseCommand("key-new"),
	}
}

func (cmd *GenerateKeyCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return xerrors.Errorf("failed to initialize command: %w", err)
	}

	var priv key.Privatekey
	if len(cmd.Type) < 1 {
		cmd.Type = btc
	} else if !IsValidKeyType(cmd.Type) {
		return xerrors.Errorf("unknown key type, %q", cmd.Type)
	} else if i := GenerateKey(cmd.Type); i == nil {
		return xerrors.Errorf("failed to generate key, %q", cmd.Type)
	} else {
		priv = i
	}

	if cmd.JSON {
		cmd.pretty(cmd.Pretty, map[string]interface{}{
			"privatekey": map[string]interface{}{
				"hint": priv.Hint(),
				"key":  priv.String(),
			},
			"publickey": map[string]interface{}{
				"hint": priv.Publickey().Hint(),
				"key":  priv.Publickey().String(),
			},
		})
	} else {
		cmd.print("      hint: %s", priv.Hint())
		cmd.print("privatekey: %s", priv.String())
		cmd.print(" publickey: %s", priv.Publickey().String())
	}

	return nil
}
