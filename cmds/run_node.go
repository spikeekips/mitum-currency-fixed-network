package cmds

import (
	"context"
	"encoding/json"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum-currency/digest"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/isaac"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"golang.org/x/xerrors"
)

var RunCommandProcesses []pm.Process

var RunCommandHooks = func(cmd *RunCommand) []pm.Hook {
	return []pm.Hook{
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameStorage,
			"set_storage", cmd.hookLoadStorage).SetOverride(true),
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameStorage,
			"load_genesis_account", cmd.hookLoadGenesisAccount).SetOverride(true),
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameNetwork,
			"set_currency_network_handlers", cmd.hookSetNetworkHandlers).SetOverride(true),
		pm.NewHook(pm.HookPrefixPre, process.ProcessNameProposalProcessor,
			"apply_fee", cmd.hookApplyFee).SetOverride(true),
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
	storage        storage.Storage
	genesisAccount *currency.Account
	genesisBalance *currency.Amount
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

	return cmd, nil
}

func (cmd *RunCommand) hookLoadStorage(ctx context.Context) (context.Context, error) {
	cmd.storage = (storage.Storage)(nil)
	if err := process.LoadStorageContextValue(ctx, &cmd.storage); err != nil {
		return ctx, err
	}

	return ctx, nil
}

func (cmd *RunCommand) hookSetStateHandler(ctx context.Context) (context.Context, error) {
	var cs *isaac.ConsensusStates
	if err := process.LoadConsensusStatesContextValue(ctx, &cs); err != nil {
		return ctx, err
	}

	var di *digest.Digester
	if err := LoadDigesterContextValue(ctx, &di); err != nil {
		if !xerrors.Is(err, config.ContextValueNotFoundError) {
			return ctx, err
		}
	}

	if cs := cs.StateHandler(base.StateConsensus); cs != nil {
		cs.(*isaac.StateConsensusHandler).WhenBlockSaved(cmd.whenBlockSaved(di))
	}
	if cs := cs.StateHandler(base.StateSyncing); cs != nil {
		cs.(*isaac.StateSyncingHandler).WhenBlockSaved(cmd.whenBlockSaved(di))
	}

	return ctx, nil
}

func (cmd *RunCommand) hookLoadGenesisAccount(ctx context.Context) (context.Context, error) {
	cmd.Log().Debug().Msg("tryingo to load genesis info")

	var enc *jsonenc.Encoder
	if err := config.LoadJSONEncoderContextValue(ctx, &enc); err != nil {
		return ctx, err
	}

	var st storage.Storage
	if err := process.LoadStorageContextValue(ctx, &st); err != nil {
		return ctx, err
	}

	var blockFS *storage.BlockFS
	if err := process.LoadBlockFSContextValue(ctx, &blockFS); err != nil {
		return nil, err
	}

	var ga *currency.Account
	var gb *currency.Amount
	if ac, ab, exists, err := cmd.loadGenesisAccount(enc, st, blockFS); err != nil {
		return ctx, err
	} else if exists {
		ga = &ac
		gb = &ab
	}

	if ga == nil || gb == nil {
		cmd.Log().Debug().Msg("genesis info not loaded")
	} else {
		cmd.genesisAccount = ga
		cmd.genesisBalance = gb

		cmd.Log().Debug().
			Interface("account", *cmd.genesisAccount).Interface("balance", *cmd.genesisBalance).
			Msg("genesis info loaded")
	}

	return ctx, nil
}

func (cmd *RunCommand) whenBlockSaved(di *digest.Digester) func([]block.Block) {
	return func(blocks []block.Block) {
		if err := cmd.checkGenesisInfo(blocks); err != nil {
			cmd.Log().Error().Err(err).Msg("failed to check genesis account info")

			return
		}

		if di != nil {
			go func() {
				di.Digest(blocks)
			}()
		}
	}
}

func (cmd *RunCommand) checkGenesisInfo(blocks []block.Block) error {
	if cmd.genesisBalance != nil && cmd.genesisBalance.Compare(currency.ZeroAmount) > 0 {
		return nil
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
		return nil
	}

	cmd.Log().Debug().Msg("trying to find genesis block")

	if ga, gb, err := cmd.saveGenesisAccountInfo(cmd.storage, genesisBlock); err != nil {
		cmd.Log().Error().Err(err).Msg("failed to save genesis account to node info")

		return err
	} else {
		cmd.genesisAccount = &ga
		cmd.genesisBalance = &gb

		return nil
	}
}

func (cmd *RunCommand) hookSetNetworkHandlers(ctx context.Context) (context.Context, error) {
	var nt network.Server
	if err := process.LoadNetworkContextValue(ctx, &nt); err != nil {
		return ctx, err
	}

	var design FeeDesign
	var fa currency.FeeAmount
	switch err := LoadFeeDesignContextValue(ctx, &design); {
	case err != nil:
		return ctx, err
	case design.FeeAmount == nil:
		return ctx, xerrors.Errorf("empty fee amount")
	default:
		fa = design.FeeAmount
	}

	nt.SetNodeInfoHandler(NodeInfoHandler(
		nt.NodeInfoHandler(),
		fa,
		func() *currency.Account {
			return cmd.genesisAccount
		},
		func() *currency.Amount {
			return cmd.genesisBalance
		},
	))

	return ctx, nil
}

func (cmd *RunCommand) hookApplyFee(ctx context.Context) (context.Context, error) {
	var design FeeDesign
	switch err := LoadFeeDesignContextValue(ctx, &design); {
	case err != nil:
		return ctx, err
	case design.FeeAmount == nil:
		return ctx, xerrors.Errorf("empty fee amount")
	case design.ReceiverFunc == nil:
		return ctx, xerrors.Errorf("empty fee receiver func")
	}

	if c, err := initializeProposalProcessor(
		ctx,
		currency.NewOperationProcessor(design.FeeAmount, design.ReceiverFunc),
	); err != nil {
		return ctx, err
	} else {
		ctx = c
	}

	cmd.Log().Debug().Interface("fee_amount", json.RawMessage([]byte(design.FeeAmount.Verbose()))).Msg("fee applied")

	return ctx, nil
}

func (cmd *RunCommand) hookDigestAPIHandlers(ctx context.Context) (context.Context, error) {
	var conf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
		return nil, err
	}

	var design DigestDesign
	if err := LoadDigestDesignContextValue(ctx, &design); err != nil {
		if xerrors.Is(err, config.ContextValueNotFoundError) {
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
	var local *isaac.Local
	if err := process.LoadLocalContextValue(ctx, &local); err != nil {
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

	rns := make([]network.Node, local.Nodes().Len()+1)
	// TODO create new local network channel for remote digest,
	rns[0] = local.Node()

	if local.Nodes().Len() > 0 { // remote nodes
		var i int = 1
		local.Nodes().Traverse(func(n network.Node) bool {
			rns[i] = n
			i++

			return true
		})
	}

	handlers := digest.NewHandlers(conf.NetworkID(), encs, jenc, st, cache).
		SetNodeInfoHandler(nt.NodeInfoHandler())

	handlers = handlers.SetSend(newSendHandler(conf.Privatekey(), conf.NetworkID(), rns))

	cmd.Log().Debug().Msg("send handler attached")

	if design.RateLimiter() != nil {
		handlers = handlers.SetRateLimiter(design.RateLimiter())
	}

	return handlers, nil
}
