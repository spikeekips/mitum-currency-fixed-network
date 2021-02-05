package cmds

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"
)

type SignSealCommand struct {
	*BaseCommand
	Privatekey PrivatekeyFlag `arg:"" name:"privatekey" help:"sender's privatekey" required:""`
	NetworkID  NetworkIDFlag  `name:"network-id" help:"network-id" required:""`
	Pretty     bool           `name:"pretty" help:"pretty format"`
	Seal       FileLoad       `help:"seal" optional:""`
}

func NewSignSealCommand() SignSealCommand {
	return SignSealCommand{
		BaseCommand: NewBaseCommand("sign-seal"),
	}
}

func (cmd *SignSealCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return xerrors.Errorf("failed to initialize command: %w", err)
	}

	var sl seal.Seal
	if s, err := loadSeal(cmd.Seal.Bytes(), cmd.NetworkID.Bytes()); err != nil {
		return err
	} else {
		sl = s
	}

	cmd.Log().Debug().Hinted("seal", sl.Hash()).Msg("seal loaded")

	if sl.Signer().Equal(cmd.Privatekey.Publickey()) {
		cmd.Log().Debug().Msg("already signed")
	} else {
		if s, err := signSeal(sl, cmd.Privatekey, cmd.NetworkID.Bytes()); err != nil {
			return err
		} else {
			cmd.Log().Debug().Msg("seal signed")

			sl = s
		}
	}

	cmd.pretty(cmd.Pretty, sl)

	return nil
}
