package cmds

import (
	"net/url"

	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/logging"
)

type SendCommand struct {
	URL        *url.URL       `name:"node" help:"remote mitum url (default: ${node_url})" default:"${node_url}"` // nolint
	NetworkID  NetworkIDFlag  `name:"network-id" help:"network-id" `
	Seal       FileLoad       `help:"seal" optional:""`
	DryRun     bool           `help:"dry-run, print operation" optional:"" default:"false"`
	Pretty     bool           `name:"pretty" help:"pretty format"`
	Privatekey PrivatekeyFlag `arg:"" name:"privatekey" help:"privatekey for sign"`
}

func (cmd *SendCommand) Run(log logging.Logger) error {
	var sl seal.Seal
	if s, err := loadSeal(cmd.Seal.Bytes(), cmd.NetworkID.Bytes()); err != nil {
		return err
	} else {
		sl = s
	}

	log.Debug().Hinted("seal", sl.Hash()).Msg("seal loaded")

	if !cmd.Privatekey.Empty() {
		if s, err := signSeal(sl, cmd.Privatekey, cmd.NetworkID.Bytes()); err != nil {
			return err
		} else {
			sl = s
		}

		log.Debug().Msg("seal signed")
	}

	if cmd.DryRun {
		prettyPrint(cmd.Pretty, sl)

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
