package cmds

import (
	"github.com/pkg/errors"

	"github.com/spikeekips/mitum/util"
)

type GenerateKeyCommand struct {
	*BaseCommand
	Seed   string `name:"seed" help:"seed (default: random string)" optional:""`
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
		return errors.Wrap(err, "failed to initialize command")
	}

	priv, err := GenerateKey(cmd.Seed)
	switch {
	case err != nil:
		return err
	case priv == nil:
		return errors.Errorf("failed to generate key")
	}

	if cmd.JSON {
		PrettyPrint(cmd.Out, cmd.Pretty, map[string]interface{}{
			"privatekey": map[string]interface{}{
				"hint": priv.Hint().Type(),
				"key":  priv.String(),
			},
			"publickey": map[string]interface{}{
				"hint": priv.Publickey().Hint().Type(),
				"key":  priv.Publickey().String(),
			},
		})
	} else {
		cmd.print("      hint: %s", priv.Hint().Type())
		cmd.print("privatekey: %s", priv.String())
		cmd.print(" publickey: %s", priv.Publickey().String())
	}

	return nil
}
