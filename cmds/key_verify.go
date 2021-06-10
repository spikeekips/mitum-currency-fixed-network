package cmds

import (
	"os"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
)

type VerifyKeyCommand struct {
	*BaseCommand
	Key    StringLoad `arg:"" name:"key" help:"key" required:"true"`
	Quite  bool       `name:"quite" short:"q" help:"keep silence"`
	JSON   bool       `name:"json" help:"json output format (default: false)" optional:"" default:"false"`
	Pretty bool       `name:"pretty" help:"pretty format"`
}

func NewVerifyKeyCommand() VerifyKeyCommand {
	return VerifyKeyCommand{
		BaseCommand: NewBaseCommand("key-verify"),
	}
}

func (cmd *VerifyKeyCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return xerrors.Errorf("failed to initialize command: %w", err)
	}

	pk, err := loadKey(cmd.Key.Bytes())
	if err != nil {
		if cmd.Quite {
			os.Exit(1) // revive:disable-line:deep-exit
		}

		return err
	}

	cmd.Log().Debug().Interface("key", pk).Msg("key parsed")

	if cmd.Quite {
		return nil
	}

	var priv key.Privatekey
	var pub key.Publickey
	switch t := pk.(type) {
	case key.Privatekey:
		priv = t
		pub = t.Publickey()
	case key.Publickey:
		pub = t
	}

	if cmd.JSON {
		m := map[string]interface{}{
			"publickey": map[string]interface{}{
				"hint": pub.Hint(),
				"key":  pub.String(),
			},
		}

		if priv != nil {
			m["privtekey"] = map[string]interface{}{
				"hint": priv.Hint(),
				"key":  priv.String(),
			}
		}

		cmd.pretty(cmd.Pretty, m)

		return nil
	}

	if priv != nil {
		cmd.print("privatekey hint: %s", priv.Hint())
		cmd.print("     privatekey: %s", priv.String())
	}

	cmd.print(" publickey hint: %s", pub.Hint())
	cmd.print("      publickey: %s", pub.String())

	return nil
}
