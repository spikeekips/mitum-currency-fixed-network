package cmds

import (
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util/logging"
)

type SignSealCommand struct {
	printCommand
	Privatekey PrivatekeyFlag `arg:"" name:"privatekey" help:"sender's privatekey" required:""`
	NetworkID  NetworkIDFlag  `name:"network-id" help:"network-id" required:""`
	Pretty     bool           `name:"pretty" help:"pretty format"`
	Seal       FileLoad       `help:"seal" optional:""`
}

func (cmd *SignSealCommand) Run(log logging.Logger) error {
	var sl seal.Seal
	if s, err := loadSeal(cmd.Seal.Bytes(), cmd.NetworkID.Bytes()); err != nil {
		return err
	} else {
		sl = s
	}

	log.Debug().Hinted("seal", sl.Hash()).Msg("seal loaded")

	if sl.Signer().Equal(cmd.Privatekey.Publickey()) {
		log.Debug().Msg("already signed")
	} else {
		if s, err := signSeal(sl, cmd.Privatekey, cmd.NetworkID.Bytes()); err != nil {
			return err
		} else {
			log.Debug().Msg("seal signed")

			sl = s
		}
	}

	cmd.pretty(cmd.Pretty, sl)

	return nil
}
