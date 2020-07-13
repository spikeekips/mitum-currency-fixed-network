package cmds

import (
	"fmt"
	"net/url"
	"os"

	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/network"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
)

type SendCommand struct {
	URL        *url.URL       `name:"node url" help:"remote mitum url (default: ${node_url})" required:"" default:"${node_url}"` // nolint
	Privatekey PrivatekeyFlag `name:"privatekey" help:"privatekey for sign"`
	NetworkID  string         `name:"network-id" help:"network-id" `
	DryRun     bool           `help:"dry-run, print operation" optional:"" default:"false"`
	Seal       string         `help:"seal" optional:"" type:"existingfile"`
}

func (cmd *SendCommand) Run(log logging.Logger) error {
	var sl seal.Seal
	if s, err := loadSealFromInput(cmd.Seal); err != nil {
		return err
	} else {
		sl = s
	}
	log.Debug().Hinted("seal", sl.Hash()).Msg("seal loaded")

	if !cmd.Privatekey.Empty() {
		if s, err := signSeal(sl, cmd.Privatekey, []byte(cmd.NetworkID)); err != nil {
			return err
		} else {
			sl = s
		}

		log.Debug().Msg("seal signed")
	}

	if cmd.DryRun {
		_, _ = fmt.Fprintln(os.Stdout, string(jsonenc.MustMarshalIndent(sl)))

		return nil
	}

	log.Debug().Msg("trying to send seal")

	if err := cmd.send(sl); err != nil {
		log.Error().Err(err).Msg("failed to send seal")

		return err
	}

	return nil
}

func (cmd *SendCommand) send(sl seal.Seal) error {
	var channel network.NetworkChannel
	if ch, err := launcher.LoadNodeChannel(cmd.URL, encs); err != nil {
		return err
	} else {
		channel = ch
	}

	return channel.SendSeal(sl)
}
