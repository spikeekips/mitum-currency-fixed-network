package cmds

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum-currency/digest"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/state"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/states"
	basicstates "github.com/spikeekips/mitum/states/basic"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/ulule/limiter/v3"
)

var RunCommandProcesses []pm.Process

var RunCommandHooks = func(cmd *RunCommand) []pm.Hook {
	return []pm.Hook{
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameDatabase,
			"set_database", HookLoadCurrencies).SetOverride(true),
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameNetwork,
			"set_currency_network_handlers", cmd.hookSetNetworkHandlers).SetOverride(true),
		pm.NewHook(pm.HookPrefixPre, process.ProcessNameProposalProcessor,
			"initialize_proposal_processor", HookInitializeProposalProcessor).SetOverride(true),
		pm.NewHook(pm.HookPrefixPost, ProcessNameDigestAPI,
			"set_digest_api_handlers", cmd.hookDigestAPIHandlers).SetOverride(true),
		pm.NewHook(pm.HookPrefixPost, ProcessNameDigester,
			"set_state_handler", cmd.hookSetStateHandler).SetOverride(true),
		pm.NewHook(pm.HookPrefixPost, ProcessNameDigester,
			HookNameDigesterFollowUp, HookDigesterFollowUp).SetOverride(true),
		pm.NewHook(pm.HookPrefixPre, ProcessNameDigestAPI,
			HookNameSetLocalChannel, HookSetLocalChannel).SetOverride(true),
	}
}

func init() {
	RunCommandProcesses = []pm.Process{
		ProcessorDigestDatabase,
		ProcessorDigester,
		ProcessorDigestAPI,
		ProcessorStartDigestAPI,
		ProcessorStartDigester,
	}
}

type RunCommand struct {
	*mitumcmds.RunCommand
	*BaseNodeCommand
}

func NewRunCommand(dryrun bool) (RunCommand, error) {
	co := mitumcmds.NewRunCommand(dryrun)
	cmd := RunCommand{
		RunCommand:      &co,
		BaseNodeCommand: NewBaseNodeCommand(co.Logging),
	}

	ps, err := cmd.BaseProcesses(co.Processes())
	if err != nil {
		return cmd, err
	}

	for i := range RunCommandProcesses {
		if err := ps.AddProcess(RunCommandProcesses[i], true); err != nil {
			return cmd, err
		}
	}

	hooks := RunCommandHooks(&cmd)
	for i := range hooks {
		if err := hooks[i].Add(ps); err != nil {
			return cmd, err
		}
	}

	_ = cmd.SetProcesses(ps)

	if err := cmd.AfterStartedHooks().Add("enter-booting-state", cmd.enteringBootingState, false); err != nil {
		return cmd, err
	}

	return cmd, nil
}

func (cmd *RunCommand) hookSetStateHandler(ctx context.Context) (context.Context, error) {
	var cs states.States
	if err := process.LoadConsensusStatesContextValue(ctx, &cs); err != nil {
		return ctx, err
	}

	var st *mongodbstorage.Database
	if err := LoadDatabaseContextValue(ctx, &st); err != nil {
		return ctx, err
	}

	var cp *currency.CurrencyPool
	if err := LoadCurrencyPoolContextValue(ctx, &cp); err != nil {
		return ctx, err
	}

	var di *digest.Digester
	if err := LoadDigesterContextValue(ctx, &di); err != nil {
		if !errors.Is(err, util.ContextValueNotFoundError) {
			return ctx, err
		}
	}

	if err := cs.BlockSavedHook().Add("mitum-currency-digest", cmd.whenBlockSaved(st, cp, di), false); err != nil {
		return ctx, err
	}

	return ctx, nil
}

func (cmd *RunCommand) whenBlockSaved(
	st *mongodbstorage.Database,
	cp *currency.CurrencyPool,
	di *digest.Digester,
) pm.ProcessFunc {
	return func(ctx context.Context) (context.Context, error) {
		var blocks []block.Block
		if err := util.LoadFromContextValue(ctx, basicstates.ContextValueBlockSaved, &blocks); err != nil {
			return ctx, err
		}

		if di != nil {
			go func() {
				di.Digest(blocks)
			}()
		}

		if err := digest.LoadCurrenciesFromDatabase(st, blocks[0].Height(), func(sta state.State) (bool, error) {
			if err := cp.Set(sta); err != nil {
				return false, err
			}
			cmd.Log().Debug().Interface("currency", sta).Msg("currency updated from mitum database")

			return true, nil
		}); err != nil {
			cmd.Log().Error().Err(err).Msg("failed to load currency designs from database")
		}

		return ctx, nil
	}
}

func (*RunCommand) hookSetNetworkHandlers(ctx context.Context) (context.Context, error) {
	var nt network.Server
	if err := process.LoadNetworkContextValue(ctx, &nt); err != nil {
		return ctx, err
	}

	nt.SetNodeInfoHandler(NodeInfoHandler(
		nt.NodeInfoHandler(),
	))

	return ctx, nil
}

func (cmd *RunCommand) hookDigestAPIHandlers(ctx context.Context) (context.Context, error) {
	var conf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
		return nil, err
	}

	var design DigestDesign
	if err := LoadDigestDesignContextValue(ctx, &design); err != nil {
		if errors.Is(err, util.ContextValueNotFoundError) {
			return ctx, nil
		}

		return nil, err
	}

	cache, err := cmd.loadCache(ctx, design)
	if err != nil {
		return ctx, err
	}

	handlers, err := cmd.setDigestHandlers(ctx, conf, design, cache)
	if err != nil {
		return ctx, err
	}
	_ = handlers.SetLogging(cmd.Logging)

	if err := handlers.Initialize(); err != nil {
		return ctx, err
	}

	var dnt *digest.HTTP2Server
	if err := LoadDigestNetworkContextValue(ctx, &dnt); err != nil {
		return ctx, err
	}
	dnt.SetRouter(handlers.Router())

	return ctx, nil
}

func (cmd *RunCommand) loadCache(_ context.Context, design DigestDesign) (digest.Cache, error) {
	c, err := digest.NewCacheFromURI(design.Cache().String())
	if err != nil {
		cmd.Log().Error().Err(err).Str("cache", design.Cache().String()).Msg("failed to connect cache server")
		cmd.Log().Warn().Msg("instead of remote cache server, internal mem cache can be available, `memory://`")

		return nil, err
	}
	return c, nil
}

func (cmd *RunCommand) setDigestHandlers(
	ctx context.Context,
	conf config.LocalNode,
	design DigestDesign,
	cache digest.Cache,
) (*digest.Handlers, error) {
	var nt network.Server
	if err := process.LoadNetworkContextValue(ctx, &nt); err != nil {
		return nil, err
	}

	var st *digest.Database
	if err := LoadDigestDatabaseContextValue(ctx, &st); err != nil {
		return nil, err
	}

	var cp *currency.CurrencyPool
	if err := LoadCurrencyPoolContextValue(ctx, &cp); err != nil {
		return nil, err
	}

	handlers := digest.NewHandlers(conf.NetworkID(), encs, jenc, st, cache, cp).
		SetNodeInfoHandler(nt.NodeInfoHandler())

	i, err := cmd.setDigestSendHandler(ctx, conf, handlers)
	if err != nil {
		return nil, err
	}
	handlers = i

	if nc := design.Network(); nc != nil && nc.RateLimit() != nil {
		if _, err := cmd.attachDigestRateLimit(ctx, handlers, nc.RateLimit()); err != nil {
			return nil, err
		}
	}

	return handlers, nil
}

func (*RunCommand) enteringBootingState(ctx context.Context) (context.Context, error) {
	var cs states.States
	var bcs *basicstates.States
	if err := process.LoadConsensusStatesContextValue(ctx, &cs); err != nil {
		return ctx, err
	} else if i, ok := cs.(*basicstates.States); !ok {
		return ctx, errors.Errorf("States not *basicstates.States, %T", cs)
	} else {
		bcs = i
	}

	if err := bcs.SwitchState(basicstates.NewStateSwitchContext(base.StateStopped, base.StateBooting)); err != nil {
		return ctx, err
	}

	return ctx, nil
}

func (*RunCommand) attachDigestRateLimit(
	ctx context.Context,
	handlers *digest.Handlers,
	conf config.RateLimit,
) (context.Context, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	rules := conf.Rules()

	handlerMap := map[string][]process.RateLimitRule{}
	for i := range rules {
		r := rules[i]

		rs := r.Rules()
		for j := range rs {
			prefix, found := digest.RateLimitHandlerMap[j]
			if !found {
				return ctx, errors.Errorf("handler, %q for digest ratelimit not found", j)
			}

			log.Log().Debug().
				Str("handler", j).
				Str("prefix", prefix).
				Str("target", r.Target()).
				Str("limit", fmt.Sprintf("%d/%s", rs[j].Limit, rs[j].Period.String())).
				Msg("found ratelimit of handler")

			handlerMap[prefix] = append(handlerMap[prefix], process.NewRateLimiterRule(r.IPNet(), rs[j]))
		}
	}

	var store limiter.Store
	if conf.Cache() != nil {
		i, err := quicnetwork.RateLimitStoreFromURI(conf.Cache().String())
		if err != nil {
			return ctx, err
		}
		log.Log().Debug().Str("store", conf.Cache().String()).Msg("ratelimit store created")

		store = i
	}

	_ = handlers.SetRateLimit(handlerMap, store)

	return ctx, nil
}

func (cmd *RunCommand) setDigestSendHandler(
	ctx context.Context,
	conf config.LocalNode,
	handlers *digest.Handlers,
) (*digest.Handlers, error) {
	var nodepool *network.Nodepool
	if err := process.LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return nil, err
	}

	var suffrage base.Suffrage
	if err := process.LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return nil, err
	}

	handlers = handlers.SetSend(NewSendHandler(conf.Privatekey(), conf.NetworkID(), func() ([]network.Channel, error) {
		remotes := suffrage.Nodes()

		var chs []network.Channel
		for i := range remotes {
			s := remotes[i]
			_, ch, found := nodepool.Node(s)
			switch {
			case !found:
				return nil, errors.Errorf("suffrage node, %q not found in nodepool", s)
			case ch == nil:
				continue
			default:
				chs = append(chs, ch)
			}
		}

		return chs, nil
	}))

	cmd.Log().Debug().Msg("send handler attached")

	return handlers, nil
}
