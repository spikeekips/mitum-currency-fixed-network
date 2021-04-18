package cmds

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/alecthomas/kong"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/seal"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
)

var SendVars = kong.Vars{
	"node_url": "quic://localhost:54321",
}

type SendCommand struct {
	*BaseCommand
	URL        []*url.URL              `name:"node" help:"remote mitum url (default: ${node_url})" default:"${node_url}"` // nolint
	NetworkID  mitumcmds.NetworkIDFlag `name:"network-id" help:"network-id" `
	Seal       FileLoad                `help:"seal" optional:""`
	DryRun     bool                    `help:"dry-run, print operation" optional:"" default:"false"`
	Pretty     bool                    `name:"pretty" help:"pretty format"`
	Privatekey PrivatekeyFlag          `arg:"" name:"privatekey" help:"privatekey for sign"`
	Timeout    time.Duration           `name:"timeout" help:"timeout; default: 5s"`
	TLSInscure bool                    `name:"tls-insecure" help:"allow inseucre TLS connection; default is false"`
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

	if cmd.Timeout < 1 {
		cmd.Timeout = time.Second * 5
	}

	var sl seal.Seal
	if s, err := loadSeal(cmd.Seal.Bytes(), cmd.NetworkID.NetworkID()); err != nil {
		return err
	} else {
		sl = s
	}

	cmd.Log().Debug().Hinted("seal", sl.Hash()).Msg("seal loaded")

	if !cmd.Privatekey.Empty() {
		if s, err := signSeal(sl, cmd.Privatekey, cmd.NetworkID.NetworkID()); err != nil {
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
	var urls []*url.URL
	founds := map[string]struct{}{}
	for i := range cmd.URL {
		u := cmd.URL[i]
		if _, found := founds[u.String()]; found {
			continue
		} else {
			founds[u.String()] = struct{}{}
			urls = append(urls, u)
		}
	}

	if len(urls) < 1 {
		return xerrors.Errorf("empty node urls")
	}

	channels := make([]network.Channel, len(urls))
	for i := range urls {
		if ch, err := process.LoadNodeChannel(urls[i], encs, cmd.Timeout, cmd.TLSInscure); err != nil {
			return err
		} else {
			channels[i] = ch
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(channels))

	errchan := make(chan error, len(channels))
	for i := range channels {
		go func(channel network.Channel) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), cmd.Timeout)
			defer cancel()

			errchan <- channel.SendSeal(ctx, sl)
		}(channels[i])
	}
	wg.Wait()
	close(errchan)

	for err := range errchan {
		if err != nil {
			return err
		}
	}

	return nil
}
