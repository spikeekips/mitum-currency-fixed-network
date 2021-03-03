package cmds

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/isaac"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/states"
	basicstates "github.com/spikeekips/mitum/states/basic"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum-currency/digest"
)

var RunCommandProcesses []pm.Process

var RunCommandHooks = func(cmd *RunCommand) []pm.Hook {
	return []pm.Hook{
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameStorage,
			"set_storage", cmd.hookLoadCurrencies).SetOverride(true),
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameNetwork,
			"set_currency_network_handlers", cmd.hookSetNetworkHandlers).SetOverride(true),
		pm.NewHook(pm.HookPrefixPre, process.ProcessNameProposalProcessor,
			"initialize_proposal_processor", cmd.hookInitializeProposalProcessor).SetOverride(true),
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
		ProcessorDigestStorage,
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

	ps := co.Processes()
	if i, err := cmd.BaseProcesses(ps); err != nil {
		return cmd, err
	} else {
		ps = i
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

func (cmd *RunCommand) hookLoadCurrencies(ctx context.Context) (context.Context, error) {
	cmd.Log().Debug().Msg("loading currencies from mitum storage")

	var st *mongodbstorage.Storage
	if err := LoadStorageContextValue(ctx, &st); err != nil {
		return ctx, err
	}

	cp := currency.NewCurrencyPool()

	if err := digest.LoadCurrenciesFromStorage(st, base.NilHeight, func(sta state.State) (bool, error) {
		if err := cp.Set(sta); err != nil {
			return false, err
		} else {
			cmd.Log().Debug().Interface("currency", sta).Msg("currency loaded from mitum storage")

			return true, nil
		}
	}); err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, ContextValueCurrencyPool, cp), nil
}

func (cmd *RunCommand) hookSetStateHandler(ctx context.Context) (context.Context, error) {
	var cs states.States
	if err := process.LoadConsensusStatesContextValue(ctx, &cs); err != nil {
		return ctx, err
	}

	var st *mongodbstorage.Storage
	if err := LoadStorageContextValue(ctx, &st); err != nil {
		return ctx, err
	}

	var cp *currency.CurrencyPool
	if err := LoadCurrencyPoolContextValue(ctx, &cp); err != nil {
		return ctx, err
	}

	var di *digest.Digester
	if err := LoadDigesterContextValue(ctx, &di); err != nil {
		if !xerrors.Is(err, util.ContextValueNotFoundError) {
			return ctx, err
		}
	}

	if err := cs.BlockSavedHook().Add("mitum-currency-digest", cmd.whenBlockSaved(st, cp, di), false); err != nil {
		return ctx, err
	}

	return ctx, nil
}

func (cmd *RunCommand) whenBlockSaved(
	st *mongodbstorage.Storage,
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

		if err := digest.LoadCurrenciesFromStorage(st, blocks[0].Height(), func(sta state.State) (bool, error) {
			if err := cp.Set(sta); err != nil {
				return false, err
			} else {
				cmd.Log().Debug().Interface("currency", sta).Msg("currency updated from mitum storage")

				return true, nil
			}
		}); err != nil {
			cmd.Log().Error().Err(err).Msg("failed to load currency designs from storage")
		}

		return ctx, nil
	}
}

func (cmd *RunCommand) hookSetNetworkHandlers(ctx context.Context) (context.Context, error) {
	var nt network.Server
	if err := process.LoadNetworkContextValue(ctx, &nt); err != nil {
		return ctx, err
	}

	nt.SetNodeInfoHandler(NodeInfoHandler(
		nt.NodeInfoHandler(),
	))

	return ctx, nil
}

func (cmd *RunCommand) hookInitializeProposalProcessor(ctx context.Context) (context.Context, error) {
	var local *network.LocalNode
	if err := process.LoadLocalNodeContextValue(ctx, &local); err != nil {
		return ctx, err
	}

	var suffrage base.Suffrage
	if err := process.LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return ctx, err
	}

	var policy *isaac.LocalPolicy
	if err := process.LoadPolicyContextValue(ctx, &policy); err != nil {
		return ctx, err
	}

	var nodepool *network.Nodepool
	if err := process.LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return ctx, err
	}

	var cp *currency.CurrencyPool
	if err := LoadCurrencyPoolContextValue(ctx, &cp); err != nil {
		return ctx, err
	}

	if opr, err := cmd.attachProposalProcessor(local, policy, nodepool, suffrage, cp); err != nil {
		return ctx, err
	} else {
		return initializeProposalProcessor(ctx, opr)
	}
}

func (cmd *RunCommand) attachProposalProcessor(
	local *network.LocalNode,
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	cp *currency.CurrencyPool,
) (*currency.OperationProcessor, error) {
	opr := currency.NewOperationProcessor(cp)
	if _, err := opr.SetProcessor(currency.CreateAccounts{}, currency.NewCreateAccountsProcessor(cp)); err != nil {
		return nil, err
	} else if _, err := opr.SetProcessor(currency.KeyUpdater{}, currency.NewKeyUpdaterProcessor(cp)); err != nil {
		return nil, err
	} else if _, err := opr.SetProcessor(currency.Transfers{}, currency.NewTransfersProcessor(cp)); err != nil {
		return nil, err
	}

	var threshold base.Threshold
	if i, err := base.NewThreshold(uint(len(suffrage.Nodes())), policy.ThresholdRatio()); err != nil {
		return nil, err
	} else {
		threshold = i
	}

	pubs := make([]key.Publickey, len(suffrage.Nodes()))
	pubs[0] = local.Publickey()
	var i int = 1
	nodepool.Traverse(func(n network.Node) bool {
		if !suffrage.IsInside(n.Address()) {
			return true
		}

		pubs[i] = n.Publickey()
		i++

		return true
	})

	if _, err := opr.SetProcessor(currency.CurrencyRegister{},
		currency.NewCurrencyRegisterProcessor(cp, pubs, threshold),
	); err != nil {
		return nil, err
	}

	if _, err := opr.SetProcessor(currency.CurrencyPolicyUpdater{},
		currency.NewCurrencyPolicyUpdaterProcessor(cp, pubs, threshold),
	); err != nil {
		return nil, err
	}

	return opr, nil
}

func (cmd *RunCommand) hookDigestAPIHandlers(ctx context.Context) (context.Context, error) {
	var conf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
		return nil, err
	}

	var design DigestDesign
	if err := LoadDigestDesignContextValue(ctx, &design); err != nil {
		if xerrors.Is(err, util.ContextValueNotFoundError) {
			return ctx, nil
		}

		return nil, err
	}

	var cache digest.Cache
	if i, err := cmd.loadCache(ctx, design); err != nil {
		return ctx, err
	} else {
		cache = i
	}

	var handlers *digest.Handlers
	if i, err := cmd.setDigestHandlers(ctx, conf, design, cache); err != nil {
		return ctx, err
	} else if err := i.Initialize(); err != nil {
		return ctx, err
	} else {
		handlers = i
	}

	_ = handlers.SetLogger(cmd.Log())

	var dnt *digest.HTTP2Server
	if err := LoadDigestNetworkContextValue(ctx, &dnt); err != nil {
		return ctx, err
	} else {
		dnt.SetHandler(handlers.Handler())

		return ctx, nil
	}
}

func (cmd *RunCommand) loadCache(_ context.Context, design DigestDesign) (digest.Cache, error) {
	if c, err := digest.NewCacheFromURI(design.Cache().String()); err != nil {
		cmd.Log().Error().Err(err).Str("cache", design.Cache().String()).Msg("failed to connect cache server")
		cmd.Log().Warn().Msg("instead of remote cache server, internal mem cache can be available, `memory://`")

		return nil, err
	} else {
		return c, nil
	}
}

func (cmd *RunCommand) setDigestHandlers(
	ctx context.Context,
	conf config.LocalNode,
	design DigestDesign,
	cache digest.Cache,
) (*digest.Handlers, error) {
	var local *network.LocalNode
	if err := process.LoadLocalNodeContextValue(ctx, &local); err != nil {
		return nil, err
	}

	var nodepool *network.Nodepool
	if err := process.LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return nil, err
	}

	var nt network.Server
	if err := process.LoadNetworkContextValue(ctx, &nt); err != nil {
		return nil, err
	}

	var st *digest.Storage
	if err := LoadDigestStorageContextValue(ctx, &st); err != nil {
		return nil, err
	}

	var cp *currency.CurrencyPool
	if err := LoadCurrencyPoolContextValue(ctx, &cp); err != nil {
		return nil, err
	}

	rns := make([]network.Node, nodepool.Len()+1)
	// TODO create new local network channel for remote digest,
	rns[0] = local

	if nodepool.Len() > 0 { // remote nodes
		var i int = 1
		nodepool.Traverse(func(n network.Node) bool {
			rns[i] = n
			i++

			return true
		})
	}

	handlers := digest.NewHandlers(conf.NetworkID(), encs, jenc, st, cache, cp).
		SetNodeInfoHandler(nt.NodeInfoHandler())

	handlers = handlers.SetSend(newSendHandler(conf.Privatekey(), conf.NetworkID(), rns))

	cmd.Log().Debug().Msg("send handler attached")

	if design.RateLimiter() != nil {
		handlers = handlers.SetRateLimiter(design.RateLimiter())
	}

	return handlers, nil
}

func (cmd *RunCommand) enteringBootingState(ctx context.Context) (context.Context, error) {
	var cs states.States
	var bcs *basicstates.States
	if err := process.LoadConsensusStatesContextValue(ctx, &cs); err != nil {
		return ctx, err
	} else if i, ok := cs.(*basicstates.States); !ok {
		return ctx, xerrors.Errorf("States not *basicstates.States, %T", cs)
	} else {
		bcs = i
	}

	if err := bcs.SwitchState(basicstates.NewStateSwitchContext(base.StateStopped, base.StateBooting)); err != nil {
		return ctx, err
	}

	return ctx, nil
}
