package cmds

import (
	"net/url"

	"github.com/alecthomas/kong"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
)

var SendVars = kong.Vars{
	"node_url": "https://localhost",
}

type SendCommand struct {
	*BaseCommand
	URL        *url.URL       `name:"node" help:"remote mitum url (default: ${node_url})" default:"${node_url}"` // nolint
	NetworkID  NetworkIDFlag  `name:"network-id" help:"network-id" `
	Seal       FileLoad       `help:"seal" optional:""`
	DryRun     bool           `help:"dry-run, print operation" optional:"" default:"false"`
	Pretty     bool           `name:"pretty" help:"pretty format"`
	Privatekey PrivatekeyFlag `arg:"" name:"privatekey" help:"privatekey for sign"`
}

func NewSendCommand() SendCommand {
	return SendCommand{
		BaseCommand: NewBaseCommand("send-seal"),
	}
}

func (cmd *SendCommand) Run(version util.Version) error {
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

	if !cmd.Privatekey.Empty() {
		if s, err := signSeal(sl, cmd.Privatekey, cmd.NetworkID.Bytes()); err != nil {
			return err
		} else {
			sl = s
		}

		cmd.Log().Debug().Msg("seal signed")
	}

	cmd.pretty(cmd.Pretty, sl)

	if cmd.DryRun {
		return nil
	}

	cmd.Log().Info().Msg("trying to send seal")

	if err := cmd.send(sl); err != nil {
		cmd.Log().Error().Err(err).Msg("failed to send seal")

		return err
	}

	cmd.Log().Info().Msg("sent seal")

	return nil
}

func (cmd *SendCommand) send(sl seal.Seal) error {
	var channel network.Channel
	if ch, err := process.LoadNodeChannel(cmd.URL, encs); err != nil {
		return err
	} else {
		channel = ch
	}

	return channel.SendSeal(sl)
}
