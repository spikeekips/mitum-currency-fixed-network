package cmds

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"

	"github.com/spikeekips/mitum-currency/currency"
)

var BaseNodeCommandHooks = func(cmd *BaseNodeCommand) []pm.Hook {
	return []pm.Hook{
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameEncoders,
			process.HookNameAddHinters, process.HookAddHinters(Hinters)).
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

func initializeProposalProcessor(ctx context.Context, opr *currency.OperationProcessor) (context.Context, error) {
	var oprs *hint.Hintmap
	if err := process.LoadOperationProcessorsContextValue(ctx, &oprs); err != nil {
		if !xerrors.Is(err, config.ContextValueNotFoundError) {
			return ctx, err
		}
	}

	if oprs == nil {
		oprs = hint.NewHintmap()

		ctx = context.WithValue(ctx, process.ContextValueOperationProcessors, oprs)
	}

	for _, hinter := range []hint.Hinter{
		currency.CreateAccounts{},
		currency.KeyUpdater{},
		currency.Transfers{},
		currency.CurrencyPolicyUpdater{},
		currency.CurrencyRegister{},
	} {
		if err := oprs.Add(hinter, opr); err != nil {
			return ctx, err
		}
	}

	return context.WithValue(ctx, process.ContextValueOperationProcessors, oprs), nil
}

func (cmd *BaseNodeCommand) hookLoadDigestConfig(ctx context.Context) (context.Context, error) {
	var source []byte
	if err := process.LoadConfigSourceContextValue(ctx, &source); err != nil {
		return ctx, err
	}

	var sourceType string
	if err := process.LoadConfigSourceTypeContextValue(ctx, &sourceType); err != nil {
		return ctx, err
	} else if sourceType != "yaml" {
		return ctx, xerrors.Errorf("unknown source type, %q", sourceType)
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

	design := *m.Digest

	if design.Network() != nil {
		if design.Network().URL() == nil {
			if err := design.Network().SetURL(DefaultDigestURL); err != nil {
				return ctx, err
			}
		}
		if design.Network().Bind() == nil {
			if err := design.Network().SetBind(DefaultDigestBind); err != nil {
				return ctx, err
			}
		}
	}

	if design.Network() == nil {
		cmd.Log().Debug().Msg("empty digest network config")
	}

	return context.WithValue(ctx, ContextValueDigestDesign, design), nil
}

func (cmd *BaseNodeCommand) hookValidateDigestConfig(ctx context.Context) (context.Context, error) {
	var conf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
		return ctx, err
	}

	var design DigestDesign
	if err := LoadDigestDesignContextValue(ctx, &design); err != nil {
		if xerrors.Is(err, config.ContextValueNotFoundError) {
			return ctx, nil
		}

		return ctx, err
	}

	if i, err := cmd.validateDigestConfigNetwork(ctx, conf, design); err != nil {
		return ctx, err
	} else {
		ctx = i
	}

	return ctx, nil
}

func (cmd *BaseNodeCommand) validateDigestConfigNetwork(
	ctx context.Context,
	conf config.LocalNode,
	design DigestDesign,
) (context.Context, error) {
	if design.Network() == nil {
		return ctx, nil
	}

	if design.Network().URL() == nil {
		return ctx, xerrors.Errorf("digest network url is missing")
	}

	a := design.Network().Bind()
	if a == nil {
		return ctx, xerrors.Errorf("digest network bind is missing")
	} else if sameBind(a, conf.Network().Bind()) {
		return ctx, xerrors.Errorf("digest bind same with mitum bind: %q", a.String())
	}

	if len(design.Network().Certs()) < 1 && design.Network().Bind().Scheme == "https" {
		if h := design.Network().Bind().Hostname(); strings.HasPrefix(h, "127.") || h == "localhost" {
			if priv, err := util.GenerateED25519Privatekey(); err != nil {
				return ctx, err
			} else if ct, err := util.GenerateTLSCerts("localhost", priv); err != nil {
				return ctx, err
			} else if err := design.Network().SetCerts(ct); err != nil {
				return ctx, err
			}
		} else {
			return ctx, xerrors.Errorf("missing certificates for https")
		}
	}

	return ctx, nil
}

func isLocal(u *url.URL) bool {
	h := u.Hostname()

	return h == "localhost" || strings.HasPrefix(h, "127.") || strings.HasPrefix(h, "0.")
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
	out io.Writer
}

func NewBaseCommand(name string) *BaseCommand {
	return &BaseCommand{
		BaseCommand: mitumcmds.NewBaseCommand(name),
		out:         os.Stdout,
	}
}

func (co *BaseCommand) pretty(pretty bool, i interface{}) {
	var b []byte
	if pretty {
		b = jsonenc.MustMarshalIndent(i)
	} else {
		b = jsonenc.MustMarshal(i)
	}

	_, _ = fmt.Fprintln(co.out, string(b))
}

func (co *BaseCommand) print(f string, a ...interface{}) {
	_, _ = fmt.Fprintf(co.out, f, a...)
	_, _ = fmt.Fprintln(co.out)
}

func hookVerboseConfig(ctx context.Context) (context.Context, error) {
	var conf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
		return ctx, err
	}

	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var dd DigestDesign
	if err := LoadDigestDesignContextValue(ctx, &dd); err != nil {
		if !xerrors.Is(err, config.ContextValueNotFoundError) {
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

	log.Debug().Interface("config", m).Msg("config loaded")

	return ctx, nil
}

type OperationFlags struct {
	Privatekey PrivatekeyFlag `arg:"" name:"privatekey" help:"privatekey to sign operation" required:""`
	Token      string         `help:"token for operation" optional:""`
	NetworkID  NetworkIDFlag  `name:"network-id" help:"network-id" required:""`
	Memo       string         `name:"memo" help:"memo"`
	Pretty     bool           `name:"pretty" help:"pretty format"`
}

func (op *OperationFlags) IsValid([]byte) error {
	if len(op.Token) < 1 {
		op.Token = localtime.String(localtime.Now())
	}

	return nil
}
