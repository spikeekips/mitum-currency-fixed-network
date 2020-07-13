package cmds

import (
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util/logging"
)

type SignSealCommand struct {
	Privatekey PrivatekeyFlag `arg:"" name:"privatekey" help:"sender's privatekey" required:""`
	NetworkID  string         `name:"network-id" help:"network-id" required:""`
	Pretty     bool           `name:"pretty" help:"pretty format"`
	Seal       string         `help:"seal" optional:"" type:"existingfile"`
}

func (cmd *SignSealCommand) Run(log logging.Logger) error {
	var fromFile bool
	var sl seal.Seal
	if s, isf, err := loadSealFromFileOrInput(cmd.Seal, []byte(cmd.NetworkID)); err != nil {
		return err
	} else {
		fromFile = isf
		sl = s
	}

	log.Debug().Bool("from_file", fromFile).Hinted("seal", sl.Hash()).Msg("seal loaded")

	if sl.Signer().Equal(cmd.Privatekey.Publickey()) {
		log.Debug().Msg("already signed")
	} else {
		if s, err := signSeal(sl, cmd.Privatekey, []byte(cmd.NetworkID)); err != nil {
			return err
		} else {
			log.Debug().Msg("seal signed")

			sl = s
		}
	}

	prettyPrint(cmd.Pretty, sl)

	return nil
}
