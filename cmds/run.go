package cmds

import (
	"crypto/tls"
	"fmt"
	"os"
	"time"

	"go.uber.org/automaxprocs/maxprocs"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum-currency/digest"
)

type RunCommand struct {
	BaseCommand
	*launcher.PprofFlags
	Design        FileLoad      `arg:"" name:"node design file" help:"node design file"`
	ExitAfter     time.Duration `help:"exit after the given duration (default: none)" default:"0s"`
	nr            *Launcher
	design        *NodeDesign
	digestStoarge *digest.Storage
	di            *digest.Digester
}

func (cmd *RunCommand) Run(flags *MainFlags, version util.Version, l logging.Logger) error {
	_ = cmd.BaseCommand.Run(flags, version, l)

	cmd.Log().Info().Msg("mitum-currency node started")

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

	if n, err := createLauncherFromDesign(cmd.Design, version, cmd.Log()); err != nil {
		return xerrors.Errorf("failed to create node runner: %w", err)
	} else {
		cmd.nr = n
		cmd.design = n.Design()
	}

	if err := cmd.initialize(); err != nil {
		return err
	}

	contestlib.ConnectSignal()
	defer contestlib.ExitHooks.Run()

	if err := cmd.nr.Start(); err != nil {
		return xerrors.Errorf("failed to start: %w", err)
	} else if err := cmd.startDigest(); err != nil {
		return err
	}

	select {
	case err := <-cmd.nr.ErrChan():
		return err
	case <-func(w time.Duration) <-chan time.Time {
		if w < 1 {
			ch := make(chan time.Time)
			return ch
		}

		return time.After(w)
	}(cmd.ExitAfter):
		cmd.Log().Info().Str("exit-after", cmd.ExitAfter.String()).Msg("expired, exit.")

		return nil
	}
}

func (cmd *RunCommand) prepareProposalProcessor() error {
	var fa currency.FeeAmount
	var feeReceiverFunc func() (base.Address, error)
	if cmd.design.FeeAmount == nil {
		fa = currency.NewNilFeeAmount()

		cmd.Log().Debug().Msg("fee not applied")
	} else {
		fa = cmd.design.FeeAmount

		var to base.Address
		if cmd.design.FeeReceiver != nil {
			to = cmd.design.FeeReceiver

			switch _, found, err := cmd.nr.Storage().State(currency.StateKeyAccount(to)); {
			case err != nil:
				return xerrors.Errorf("failed to find fee receiver, %v: %w", to, err)
			case !found:
				return xerrors.Errorf("fee receiver, %v does not exist", to)
			}
		} else if gac, _, exists := cmd.nr.genesisInfo(); exists {
			to = gac.Address()
		}

		if to != nil {
			feeReceiverFunc = func() (base.Address, error) {
				return to, nil
			}
		} else {
			feeReceiverFunc = func() (base.Address, error) {
				if gac, _, exists := cmd.nr.genesisInfo(); exists {
					return gac.Address(), nil
				} else {
					return nil, nil
				}
			}
		}

		cmd.Log().Debug().
			Str("fee_amount", cmd.design.FeeAmount.Verbose()).
			Interface("fee_receiver", to).Msg("fee applied")
	}

	return initlaizeProposalProcessor(
		cmd.nr.ProposalProcessor(),
		currency.NewOperationProcessor(fa, feeReceiverFunc),
	)
}

func (cmd *RunCommand) whenBlockSaved(blocks []block.Block) {
	cmd.checkGenesisInfo(blocks)

	go func() {
		cmd.di.Digest(blocks)
	}()
}

func (cmd *RunCommand) checkGenesisInfo(blocks []block.Block) {
	if _, _, exists := cmd.nr.genesisInfo(); exists {
		return
	}

	// NOTE catch genesis block
	var genesisBlock block.Block
	for _, blk := range blocks {
		if blk.Height() == base.Height(0) {
			genesisBlock = blk

			break
		}
	}

	if genesisBlock == nil {
		return
	}

	cmd.Log().Debug().Msg("trying to find genesis block")
	if ga, gb, err := saveGenesisAccountInfo(cmd.nr.Storage(), genesisBlock, cmd.Log()); err != nil {
		cmd.Log().Error().Err(err).Msg("failed to save genesis account to node info")

		cmd.nr.setGenesisInfo(currency.Account{}, currency.ZeroAmount)
	} else {
		cmd.nr.setGenesisInfo(ga, gb)
	}
}

func (cmd *RunCommand) initialize() error {
	if err := cmd.nr.Initialize(); err != nil {
		return xerrors.Errorf("failed to generate node from design: %w", err)
	} else if err := cmd.prepareProposalProcessor(); err != nil {
		return err
	}

	if cs := cmd.nr.ConsensusStates().StateHandler(base.StateConsensus); cs != nil {
		cs.(*isaac.StateConsensusHandler).WhenBlockSaved(cmd.whenBlockSaved)
	}
	if cs := cmd.nr.ConsensusStates().StateHandler(base.StateSyncing); cs != nil {
		cs.(*isaac.StateSyncingHandler).WhenBlockSaved(cmd.whenBlockSaved)
	}

	if ac, ba, err := loadGenesisAccountInfo(cmd.nr.Storage(), cmd.Log()); err != nil {
		cmd.Log().Error().Err(err).Msg("failed to load genesis account info")
	} else {
		cmd.nr.setGenesisInfo(ac, ba) // NOTE set for NodeInfo
	}

	if len(cmd.design.Digest.Storage) < 1 {
		cmd.log.Debug().Msg("digest storage disabled")
	} else if st, err := loadDigestStorage(cmd.design, cmd.nr.Storage(), false); err != nil {
		return err
	} else {
		_ = st.SetLogger(cmd.log)
		cmd.digestStoarge = st
	}

	return nil
}

func (cmd *RunCommand) startDigest() error {
	if cmd.digestStoarge == nil {
		cmd.log.Debug().Msg("digester disabled")

		return nil
	}

	if err := cmd.startDigester(); err != nil {
		return err
	} else {
		cmd.log.Debug().Msg("digester started")
	}

	if err := cmd.startDigestAPI(); err != nil {
		return err
	} else {
		cmd.log.Debug().Msg("digest API started")
	}

	return nil
}

func (cmd *RunCommand) startDigester() error {
	cmd.log.Debug().Msg("start digesting")

	// check current blocks
	switch m, found, err := cmd.nr.Storage().LastManifest(); {
	case err != nil:
		return err
	case !found:
		cmd.log.Debug().Msg("last manifest not found")
	case m.Height() > cmd.digestStoarge.LastBlock():
		cmd.log.Info().
			Hinted("last_manifest", m.Height()).
			Hinted("last_block", cmd.digestStoarge.LastBlock()).
			Msg("new blocks found to digest")

		if err := cmd.digestFollowup(m.Height()); err != nil {
			cmd.log.Error().Err(err).Msg("failed to follow up")

			return err
		} else {
			cmd.log.Info().Msg("digested new blocks")
		}
	default:
		cmd.log.Info().Msg("digested blocks is up-to-dated")
	}

	cmd.di = digest.NewDigester(cmd.digestStoarge, nil)
	_ = cmd.di.SetLogger(cmd.log)

	return cmd.di.Start()
}

func (cmd *RunCommand) digestFollowup(height base.Height) error {
	if height <= cmd.digestStoarge.LastBlock() {
		return nil
	}

	lastBlock := cmd.digestStoarge.LastBlock()
	if lastBlock < base.PreGenesisHeight {
		lastBlock = base.PreGenesisHeight
	}

	for i := lastBlock; i <= height; i++ {
		if blk, err := cmd.nr.Localstate().BlockFS().Load(i); err != nil {
			return err
		} else if err := digest.DigestBlock(cmd.digestStoarge, blk); err != nil {
			return err
		}
	}

	return nil
}

func (cmd *RunCommand) startDigestAPI() error {
	if cmd.digestStoarge == nil || cmd.design.Digest.Network == nil {
		cmd.log.Debug().Msg("digest API disabled")

		return nil
	}

	var cache digest.Cache
	if mc, err := digest.NewCacheFromURI(cmd.design.Digest.Cache); err != nil {
		cmd.log.Error().Err(err).Str("cache", cmd.design.Digest.Cache).Msg("failed to connect cache server")
		cmd.log.Warn().Msg("instead of remote cache server, internal mem cache can be available, `memory://`")

		return err
	} else {
		cache = mc
	}

	cmd.log.Info().
		Str("bind", cmd.design.Digest.Network.Bind().String()).
		Str("publish", cmd.design.Digest.Network.PublishURL().String()).
		Msg("trying to start http2 server for digest API")
	var nt *digest.HTTP2Server

	var certs []tls.Certificate
	if cmd.design.Digest.Network.Bind().Scheme == "https" {
		certs = cmd.design.Digest.Network.Certs()
	}

	if sv, err := digest.NewHTTP2Server(
		cmd.design.Digest.Network.Bind().Host,
		cmd.design.Network.PublishURL().Host,
		certs,
	); err != nil {
		return err
	} else if err := sv.Initialize(); err != nil {
		return err
	} else {
		_ = sv.SetLogger(cmd.log)

		nt = sv
	}

	if handlers, err := cmd.handlers(cache); err != nil {
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

	return nil
}

func (cmd *RunCommand) handlers(cache digest.Cache) (*digest.Handlers, error) {
	handlers := digest.NewHandlers(cmd.design.NetworkID(), encs, defaultJSONEnc, cmd.digestStoarge, cache).
		SetNodeInfoHandler(cmd.nr.nodeInfoHandler)

	var rns []network.Node
	if n, err := cmd.design.NetworkNode(encs); err != nil {
		return nil, err
	} else {
		rns = append(rns, n)
	}

	if cmd.nr.Localstate().Nodes().Len() > 0 { // remote nodes
		cmd.nr.Localstate().Nodes().Traverse(func(rn network.Node) bool {
			rns = append(rns, rn)

			return true
		})
	}

	handlers = handlers.SetSend(newSendHandler(cmd.design.Privatekey(), cmd.design.NetworkID(), rns))

	cmd.log.Debug().Msg("send handler attached")

	if cmd.design.RateLimiter != nil {
		handlers = handlers.SetRateLimiter(cmd.design.RateLimiter.Limiter())
	}

	_ = handlers.SetLogger(cmd.log)

	if err := handlers.Initialize(); err != nil {
		return nil, err
	} else {
		return handlers, nil
	}
}
