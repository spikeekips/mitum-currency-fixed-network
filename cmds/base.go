package cmds

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum-currency/digest"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/isaac"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
)

const localhost = "localhost"

var BaseNodeCommandHooks = func(cmd *BaseNodeCommand) []pm.Hook {
	return []pm.Hook{
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameEncoders,
			process.HookNameAddHinters, process.HookAddHinters(Types, Hinters)).
			SetOverride(true),
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameConfig,
			"load_digest_config", cmd.hookLoadDigestConfig).
			SetOverride(true).
			SetDir(process.HookNameConfigGenesisOperations, pm.HookDirAfter),
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameConfig,
			"validate_digest_config", cmd.hookValidateDigestConfig).
			SetOverride(true).
			SetDir(process.HookNameValidateConfig, pm.HookDirAfter),
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameConfig,
			process.HookNameConfigVerbose, hookVerboseConfig).
			SetOverride(true),
	}
}

type BaseNodeCommand struct {
	*logging.Logging
}

func NewBaseNodeCommand(l *logging.Logging) *BaseNodeCommand {
	return &BaseNodeCommand{Logging: l}
}

func (cmd *BaseNodeCommand) BaseProcesses(ps *pm.Processes) (*pm.Processes, error) {
	hooks := BaseNodeCommandHooks(cmd)
	for i := range hooks {
		if err := hooks[i].Add(ps); err != nil {
			return nil, err
		}
	}

	return ps, nil
}

func HookLoadCurrencies(ctx context.Context) (context.Context, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	log.Log().Debug().Msg("loading currencies from mitum database")

	var st *mongodbstorage.Database
	if err := LoadDatabaseContextValue(ctx, &st); err != nil {
		return ctx, err
	}

	cp := currency.NewCurrencyPool()

	if err := digest.LoadCurrenciesFromDatabase(st, base.NilHeight, func(sta state.State) (bool, error) {
		if err := cp.Set(sta); err != nil {
			return false, err
		}
		log.Log().Debug().Interface("currency", sta).Msg("currency loaded from mitum database")

		return true, nil
	}); err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, ContextValueCurrencyPool, cp), nil
}

func HookInitializeProposalProcessor(ctx context.Context) (context.Context, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var suffrage base.Suffrage
	if err := process.LoadSuffrageContextValue(ctx, &suffrage); err != nil {
		return ctx, err
	}

	var nodepool *network.Nodepool
	if err := process.LoadNodepoolContextValue(ctx, &nodepool); err != nil {
		return ctx, err
	}

	if !suffrage.IsInside(nodepool.LocalNode().Address()) {
		log.Log().Debug().Msg("none-suffrage node; proposal processor will not be used")

		return ctx, nil
	}

	var policy *isaac.LocalPolicy
	if err := process.LoadPolicyContextValue(ctx, &policy); err != nil {
		return ctx, err
	}

	var cp *currency.CurrencyPool
	if err := LoadCurrencyPoolContextValue(ctx, &cp); err != nil {
		return ctx, err
	}

	opr, err := AttachProposalProcessor(policy, nodepool, suffrage, cp)
	if err != nil {
		return ctx, err
	}

	_ = opr.SetLogging(log)

	return InitializeProposalProcessor(ctx, opr)
}

func AttachProposalProcessor(
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	cp *currency.CurrencyPool,
) (*currency.OperationProcessor, error) {
	opr := currency.NewOperationProcessor(cp)
	if _, err := opr.SetProcessor(currency.CreateAccountsHinter, currency.NewCreateAccountsProcessor(cp)); err != nil {
		return nil, err
	} else if _, err := opr.SetProcessor(currency.KeyUpdaterHinter, currency.NewKeyUpdaterProcessor(cp)); err != nil {
		return nil, err
	} else if _, err := opr.SetProcessor(currency.TransfersHinter, currency.NewTransfersProcessor(cp)); err != nil {
		return nil, err
	}

	threshold, err := base.NewThreshold(uint(len(suffrage.Nodes())), policy.ThresholdRatio())
	if err != nil {
		return nil, err
	}

	suffrageNodes := suffrage.Nodes()
	pubs := make([]key.Publickey, len(suffrageNodes))
	for i := range suffrageNodes {
		n, _, found := nodepool.Node(suffrageNodes[i])
		if !found {
			return nil, errors.Errorf("suffrage node, %q not found in nodepool", suffrageNodes[i])
		}
		pubs[i] = n.Publickey()
	}

	if _, err := opr.SetProcessor(currency.CurrencyRegisterHinter,
		currency.NewCurrencyRegisterProcessor(cp, pubs, threshold),
	); err != nil {
		return nil, err
	}

	if _, err := opr.SetProcessor(currency.CurrencyPolicyUpdaterHinter,
		currency.NewCurrencyPolicyUpdaterProcessor(cp, pubs, threshold),
	); err != nil {
		return nil, err
	}

	if _, err := opr.SetProcessor(currency.SuffrageInflationHinter,
		currency.NewSuffrageInflationProcessor(cp, pubs, threshold),
	); err != nil {
		return nil, err
	}

	return opr, nil
}

func InitializeProposalProcessor(ctx context.Context, opr *currency.OperationProcessor) (context.Context, error) {
	var oprs *hint.Hintmap
	if err := process.LoadOperationProcessorsContextValue(ctx, &oprs); err != nil {
		if !errors.Is(err, util.ContextValueNotFoundError) {
			return ctx, err
		}
	}

	if oprs == nil {
		oprs = hint.NewHintmap()

		ctx = context.WithValue(ctx, process.ContextValueOperationProcessors, oprs)
	}

	for _, hinter := range []hint.Hinter{
		currency.CreateAccountsHinter,
		currency.KeyUpdaterHinter,
		currency.TransfersHinter,
		currency.CurrencyPolicyUpdaterHinter,
		currency.CurrencyRegisterHinter,
		currency.SuffrageInflationHinter,
	} {
		if err := oprs.Add(hinter, opr); err != nil {
			return ctx, err
		}
	}

	return context.WithValue(ctx, process.ContextValueOperationProcessors, oprs), nil
}

func (*BaseNodeCommand) hookLoadDigestConfig(ctx context.Context) (context.Context, error) {
	var source []byte
	if err := process.LoadConfigSourceContextValue(ctx, &source); err != nil {
		return ctx, err
	}

	var sourceType string
	if err := process.LoadConfigSourceTypeContextValue(ctx, &sourceType); err != nil {
		return ctx, err
	} else if sourceType != "yaml" {
		return ctx, errors.Errorf("unknown source type, %q", sourceType)
	}

	var m struct {
		Digest *DigestDesign
	}

	if err := yaml.Unmarshal(source, &m); err != nil {
		return ctx, err
	} else if m.Digest == nil {
		return ctx, nil
	} else if i, err := m.Digest.Set(ctx); err != nil {
		return ctx, err
	} else {
		ctx = i
	}

	return context.WithValue(ctx, ContextValueDigestDesign, *m.Digest), nil
}

func (cmd *BaseNodeCommand) hookValidateDigestConfig(ctx context.Context) (context.Context, error) {
	var conf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
		return ctx, err
	}

	var design DigestDesign
	if err := LoadDigestDesignContextValue(ctx, &design); err != nil {
		if errors.Is(err, util.ContextValueNotFoundError) {
			return ctx, nil
		}

		return ctx, err
	}

	if design.Network() != nil {
		i, err := cmd.validateDigestConfigNetwork(ctx, conf, design)
		if err != nil {
			return ctx, err
		}
		ctx = i
	}

	return ctx, nil
}

func (cmd *BaseNodeCommand) validateDigestConfigNetwork(
	ctx context.Context,
	conf config.LocalNode,
	design DigestDesign,
) (context.Context, error) {
	if design.Network().ConnInfo() == nil {
		return ctx, errors.Errorf("digest network url is missing")
	}

	a := design.Network().Bind()
	if a == nil {
		return ctx, errors.Errorf("digest network bind is missing")
	} else if sameBind(a, conf.Network().Bind()) {
		return ctx, errors.Errorf("digest bind same with mitum bind: %q", a.String())
	}

	if len(design.Network().Certs()) < 1 && design.Network().Bind().Scheme == "https" {
		if h := design.Network().Bind().Hostname(); !strings.HasPrefix(h, "127.") && h != localhost {
			return ctx, errors.Errorf("missing certificates for https")
		}

		if priv, err := util.GenerateED25519Privatekey(); err != nil {
			return ctx, err
		} else if ct, err := util.GenerateTLSCerts(localhost, priv); err != nil {
			return ctx, err
		} else if err := design.Network().SetCerts(ct); err != nil {
			return ctx, err
		}
	}

	if design.Network().RateLimit() != nil {
		i, err := cmd.validateDigestConfigNetworkRateLimit(ctx, design)
		if err != nil {
			return i, err
		}
		ctx = i
	}

	return ctx, nil
}

func (*BaseNodeCommand) validateDigestConfigNetworkRateLimit(
	ctx context.Context,
	design DigestDesign,
) (context.Context, error) {
	rcc := config.NewRateLimitChecker(ctx, design.Network().RateLimit(), nil)

	if err := util.NewChecker("config-ratelimit-checker", []util.CheckerFunc{
		rcc.Initialize,
		rcc.Check,
	}).Check(); err != nil {
		if !errors.Is(err, util.IgnoreError) {
			return ctx, err
		}
	}

	return ctx, design.Network().SetRateLimit(rcc.Config())
}

func isLocal(u *url.URL) bool {
	h := u.Hostname()

	return h == localhost || strings.HasPrefix(h, "127.") || strings.HasPrefix(h, "0.")
}

func sameBind(a, b *url.URL) bool {
	if a.Scheme != b.Scheme || a.Port() != b.Port() {
		return false
	}

	ha := a.Hostname()
	if isLocal(a) {
		ha = "127.0.0.1"
	}
	hb := b.Hostname()
	if isLocal(b) {
		hb = "127.0.0.1"
	}

	return ha == hb
}

type BaseCommand struct {
	*mitumcmds.BaseCommand
	Out io.Writer `kong:"-"`
}

func NewBaseCommand(name string) *BaseCommand {
	return &BaseCommand{
		BaseCommand: mitumcmds.NewBaseCommand(name),
		Out:         os.Stdout,
	}
}

func (co *BaseCommand) print(f string, a ...interface{}) {
	_, _ = fmt.Fprintf(co.Out, f, a...)
	_, _ = fmt.Fprintln(co.Out)
}

func hookVerboseConfig(ctx context.Context) (context.Context, error) {
	var conf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
		return ctx, err
	}

	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var dd DigestDesign
	if err := LoadDigestDesignContextValue(ctx, &dd); err != nil {
		if !errors.Is(err, util.ContextValueNotFoundError) {
			return ctx, err
		}
	}

	var m map[string]interface{}
	if b, err := jsonenc.Marshal(conf); err != nil {
		return ctx, err
	} else if err := jsonenc.Unmarshal(b, &m); err != nil {
		return ctx, err
	}

	m["digest"] = dd

	log.Log().Debug().Interface("config", m).Msg("config loaded")

	return ctx, nil
}

type OperationFlags struct {
	Privatekey PrivatekeyFlag          `arg:"" name:"privatekey" help:"privatekey to sign operation" required:"true"`
	Token      string                  `help:"token for operation" optional:""`
	NetworkID  mitumcmds.NetworkIDFlag `name:"network-id" help:"network-id" required:"true"`
	Memo       string                  `name:"memo" help:"memo"`
	Pretty     bool                    `name:"pretty" help:"pretty format"`
}

func (op *OperationFlags) IsValid([]byte) error {
	if len(op.Token) < 1 {
		op.Token = localtime.String(localtime.UTCNow())
	}

	return op.NetworkID.NetworkID().IsValid(nil)
}
