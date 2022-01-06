package cmds

import (
	"context"
	"net/url"
	"time"

	"github.com/alecthomas/kong"
	"github.com/pkg/errors"
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
	Seal       mitumcmds.FileLoad      `help:"seal" optional:""`
	DryRun     bool                    `help:"dry-run, print operation" optional:"" default:"false"`
	Pretty     bool                    `name:"pretty" help:"pretty format"`
	Privatekey PrivatekeyFlag          `arg:"" name:"privatekey" help:"privatekey for sign"`
	Timeout    time.Duration           `name:"timeout" help:"timeout; default: 5s"`
	TLSInscure bool                    `name:"tls-insecure" help:"allow inseucre TLS connection; default is false"`
	From       string                  `name:"from" help:"from conninfo; default is empty"`
}

func NewSendCommand() SendCommand {
	return SendCommand{
		BaseCommand: NewBaseCommand("send-seal"),
	}
}

func (cmd *SendCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return errors.Wrap(err, "failed to initialize command")
	}

	if cmd.Timeout < 1 {
		cmd.Timeout = time.Second * 5
	}

	sl, err := LoadSeal(cmd.Seal.Bytes(), cmd.NetworkID.NetworkID())
	if err != nil {
		return err
	}

	cmd.Log().Debug().Stringer("seal", sl.Hash()).Msg("seal loaded")

	if !cmd.Privatekey.Empty() {
		s, err := SignSeal(sl, cmd.Privatekey, cmd.NetworkID.NetworkID())
		if err != nil {
			return err
		}
		sl = s

		cmd.Log().Debug().Msg("seal signed")
	}

	PrettyPrint(cmd.Out, cmd.Pretty, sl)

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
	var urls []*url.URL // nolint:prealloc
	founds := map[string]struct{}{}
	for i := range cmd.URL {
		u := cmd.URL[i]
		if _, found := founds[u.String()]; found {
			continue
		}
		founds[u.String()] = struct{}{}
		urls = append(urls, u)
	}

	if len(urls) < 1 {
		return errors.Errorf("empty node urls")
	}

	channels := make([]network.Channel, len(urls))
	for i := range urls {
		u := urls[i]
		connInfo := network.NewHTTPConnInfo(u, cmd.TLSInscure)
		ch, err := process.LoadNodeChannel(connInfo, encs, cmd.Timeout)
		if err != nil {
			return err
		}
		channels[i] = ch
	}

	ctx, cancel := context.WithTimeout(context.Background(), cmd.Timeout)
	defer cancel()

	wk := util.NewDistributeWorker(ctx, 100, nil)
	defer wk.Close()

	go func() {
		defer wk.Done()

		for i := range channels {
			ch := channels[i]
			if err := wk.NewJob(func(ctx context.Context, _ uint64) error {
				if err := ch.SendSeal(ctx, network.NewNilConnInfo(cmd.From), sl); err != nil {
					cmd.Log().Error().Err(err).Stringer("conninfo", ch.ConnInfo()).Msg("failed to send to node")

					return err
				}

				return nil
			}); err != nil {
				return
			}
		}
	}()

	return wk.Wait()
}
