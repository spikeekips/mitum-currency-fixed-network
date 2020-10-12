package cmds

import (
	"fmt"
	"net/url"
	"os"

	"go.uber.org/automaxprocs/maxprocs"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum-currency/digest"
	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type DigestCommand struct {
	BaseCommand
	*launcher.PprofFlags
	Design FileLoad `arg:"" name:"digest design file" help:"digest design file"`
	Node   *url.URL `name:"node url" help:"mitum currency url (default: ${node_url})" required:"" default:"${node_url}"` // nolint
	design *NodeDesign
	ch     network.Channel
}

func (cmd *DigestCommand) Run(flags *MainFlags, version util.Version, l logging.Logger) error {
	_ = cmd.BaseCommand.Run(flags, version, l)

	cmd.Log().Info().Msg("mitum-currency digest server started")

	_, _ = maxprocs.Set(maxprocs.Logger(func(f string, s ...interface{}) {
		cmd.Log().Debug().Msgf(f, s...)
	}))

	if cancel, err := launcher.RunPprof(cmd.PprofFlags); err != nil {
		return err
	} else {
		contestlib.ExitHooks.Add(func() {
			if err := cancel(); err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err.Error())
			}
		})
	}

	contestlib.ConnectSignal()
	defer contestlib.ExitHooks.Run()

	switch d, err := LoadNodeDesign(cmd.Design.Bytes(), encs); {
	case err != nil:
		return err
	case d.Digest.Network == nil:
		return xerrors.Errorf("network is missing")
	default:
		cmd.design = d
	}

	cmd.log.Info().
		Str("bind", cmd.design.Digest.Network.BindString).
		Str("publish", cmd.design.Digest.Network.PublishURL().String()).
		Msg("trying to start http2 server for digest API")

	if ch, err := launcher.LoadNodeChannel(cmd.Node, encs); err != nil {
		return err
	} else if _, err := ch.NodeInfo(); err != nil {
		return err
	} else {
		cmd.ch = ch
	}

	return cmd.run()
}

func (cmd *DigestCommand) run() error {
	var st *digest.Storage
	if mst, err := launcher.LoadStorage(cmd.design.Storage, encs, nil); err != nil {
		return err
	} else if s, err := loadDigestStorage(cmd.design, mst, true); err != nil {
		return err
	} else {
		st = s
	}

	var cache digest.Cache
	if mc, err := digest.NewCacheFromURI(cmd.design.Digest.Cache); err != nil {
		cmd.log.Error().Err(err).Str("cache", cmd.design.Digest.Cache).Msg("failed to connect cache server")
		cmd.log.Warn().Msg("instead of remote cache server, internal mem cache can be available, `memory://`")

		return err
	} else {
		cache = mc
	}

	var nt *digest.HTTP2Server
	if sv, err := digest.NewHTTP2Server(
		cmd.design.Digest.Network.Bind().Host,
		cmd.design.Digest.Network.PublishURL().Host,
		cmd.design.Digest.Network.Certs(),
	); err != nil {
		return err
	} else if err := sv.Initialize(); err != nil {
		return err
	} else {
		_ = sv.SetLogger(cmd.log)

		nt = sv
	}

	if handlers, err := cmd.handlers(st, cache); err != nil {
		return err
	} else {
		nt.SetHandler(handlers.Handler())
	}

	if err := nt.Start(); err != nil {
		return err
	}

	contestlib.ExitHooks.Add(func() {
		if err := nt.Stop(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
		}
	})

	select {}
}

func (cmd *DigestCommand) nodeInfo() (network.NodeInfo, error) {
	return cmd.ch.NodeInfo()
}

func (cmd *DigestCommand) handlers(st *digest.Storage, cache digest.Cache) (*digest.Handlers, error) {
	handlers := digest.NewHandlers(cmd.design.NetworkID(), encs, defaultJSONEnc, st, cache).
		SetNodeInfoHandler(cmd.nodeInfo)
	_ = handlers.SetLogger(cmd.log)

	if len(cmd.design.Nodes) > 0 { // remote nodes
		var rns []network.Node
		for i := range cmd.design.Nodes {
			if n, err := cmd.design.Nodes[i].NetworkNode(encs); err != nil {
				return nil, err
			} else {
				rns = append(rns, n)
			}
		}

		if n, err := cmd.design.NetworkNode(encs); err != nil {
			return nil, err
		} else {
			rns = append(rns, n)
		}

		handlers = handlers.SetSend(newSendHandler(cmd.design.Privatekey(), cmd.design.NetworkID(), rns))

		cmd.log.Debug().Msg("send handler attached")
	}

	if err := handlers.Initialize(); err != nil {
		return nil, err
	}

	return handlers, nil
}
