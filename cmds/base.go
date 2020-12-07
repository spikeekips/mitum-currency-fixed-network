package cmds

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/isaac"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
)

var BaseNodeCommandHooks = func(cmd *BaseNodeCommand) []pm.Hook {
	genesisOperationHandlers := map[string]process.HookHandlerGenesisOperations{
		"genesis-account": cmd.genesisOperationsHandlerGenesisAccount,
	}
	for k, v := range process.DefaultHookHandlersGenesisOperations {
		genesisOperationHandlers[k] = v
	}

	return []pm.Hook{
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameEncoders,
			process.HookNameAddHinters, process.HookAddHinters(Hinters)).
			SetOverride(true),
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameConfig,
			process.HookNameConfigGenesisOperations, process.HookGenesisOperationFunc(genesisOperationHandlers)).
			SetOverride(true),
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameConfig,
			"load_fee_config", cmd.hookLoadFeeConfig).
			SetOverride(true).
			SetDir(process.HookNameConfigGenesisOperations, pm.HookDirAfter),
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameStorage,
			"validate_fee_config", cmd.hookValidateFeeConfig).
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

func (cmd *BaseNodeCommand) genesisOperationsHandlerGenesisAccount(
	ctx context.Context,
	m map[string]interface{},
) (operation.Operation, error) {
	var conf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
		return nil, err
	}

	var gad *GenesisAccountDesign
	if b, err := yaml.Marshal(m); err != nil {
		return nil, err
	} else if err := yaml.Unmarshal(b, &gad); err != nil {
		return nil, err
	}

	if err := gad.IsValid(nil); err != nil {
		return nil, err
	}

	if op, err := currency.NewGenesisAccount(
		conf.Privatekey(),
		gad.AccountKeys.Keys,
		gad.Balance,
		conf.NetworkID(),
	); err != nil {
		return nil, err
	} else {
		return op, nil
	}
}

func (cmd *BaseNodeCommand) saveGenesisAccountInfo(
	st storage.Storage,
	blk block.Block,
) (currency.Account, currency.Amount, error) {
	cmd.Log().Debug().Msg("trying to save genesis info")

	var ga currency.Account
	var gb currency.Amount = currency.NilAmount
	for i := range blk.States() {
		s := blk.States()[i]
		if currency.IsStateAccountKey(s.Key()) {
			if ac, err := currency.LoadStateAccountValue(s); err != nil {
				return ga, gb, err
			} else {
				ga = ac
			}
		} else if currency.IsStateBalanceKey(s.Key()) {
			if am, err := currency.StateAmountValue(s); err != nil {
				return ga, gb, err
			} else {
				gb = am
			}
		}
	}

	if ga.IsEmpty() {
		return ga, gb, xerrors.Errorf("failed to find genesis account")
	}

	if gb.Compare(currency.ZeroAmount) <= 0 {
		return ga, gb, xerrors.Errorf("failed to find genesis balance")
	}

	if b, err := jsonenc.Marshal(ga); err != nil {
		return ga, gb, xerrors.Errorf("failed to save genesis account: %w", err)
	} else if err := st.SetInfo(GenesisAccountKey, b); err != nil {
		return ga, gb, xerrors.Errorf("failed to save genesis account: %w", err)
	}

	if b, err := jsonenc.Marshal(gb); err != nil {
		return ga, gb, xerrors.Errorf("failed to save genesis balance: %w", err)
	} else if err := st.SetInfo(GenesisBalanceKey, b); err != nil {
		return ga, gb, xerrors.Errorf("failed to save genesis balance: %w", err)
	}

	cmd.Log().Debug().Msg("genesis info saved")

	return ga, gb, nil
}

func (cmd *BaseNodeCommand) loadGenesisAccount(
	enc encoder.Encoder,
	st storage.Storage,
	blockFS *storage.BlockFS,
) (currency.Account, currency.Amount, bool, error) {
	cmd.Log().Debug().Msg("tryingo to load genesis info")

	var ac currency.Account
	var ab currency.Amount = currency.NilAmount
	switch b, found, err := st.Info(GenesisAccountKey); {
	case err != nil:
		return ac, ab, false, xerrors.Errorf("failed to get genesis account: %w", err)
	case !found:
		cmd.Log().Debug().Err(err).Msg("genesis account info not found; will try to load from stoarge")
	default:
		if err := enc.Decode(b, &ac); err != nil {
			return ac, ab, false, xerrors.Errorf("failed to load genesis account for getting fee receiver: %w", err)
		}
	}

	switch b, found, err := st.Info(GenesisBalanceKey); {
	case err != nil:
		return ac, ab, false, xerrors.Errorf("failed to get genesis balance: %w", err)
	case !found:
		cmd.Log().Debug().Err(err).Msg("genesis balance not found")
	default:
		if err := enc.Decode(b, &ab); err != nil {
			return ac, ab, false, xerrors.Errorf("failed to load genesis balance: %w", err)
		}
	}

	if ab.Compare(currency.NilAmount) > 0 {
		return ac, ab, true, nil
	}

	switch a, b, found, err := cmd.loadGenesisAccountFromBlockFS(st, blockFS); {
	case err != nil:
		return ac, ab, false, err
	case !found:
		return ac, ab, false, nil
	default:
		ac = a
		ab = b
	}

	return ac, ab, true, nil
}

func (cmd *BaseNodeCommand) loadGenesisAccountFromBlockFS(
	st storage.Storage,
	blockFS *storage.BlockFS,
) (currency.Account, currency.Amount, bool, error) {
	var ac currency.Account
	var ab currency.Amount = currency.NilAmount
	if _, err := blockFS.Exists(base.Height(0)); err != nil {
		if xerrors.Is(err, storage.NotFoundError) {
			return ac, ab, false, nil
		}

		return ac, ab, false, err
	}

	if blk, err := blockFS.Load(base.Height(0)); err != nil {
		cmd.Log().Debug().Err(err).Msg("genesis account info not found in blockFS")

		return ac, ab, false, err
	} else if c, b, err := cmd.saveGenesisAccountInfo(st, blk); err != nil {
		return ac, ab, false, err
	} else {
		ac = c
		ab = b

		return ac, ab, true, nil
	}
}

func (cmd *BaseNodeCommand) hookLoadFeeConfig(ctx context.Context) (context.Context, error) {
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
		Fee FeeDesign
	}

	if err := yaml.Unmarshal(source, &m); err != nil {
		return ctx, err
	} else if m.Fee.FeeAmount == nil {
		m.Fee.FeeAmount = currency.NewNilFeeAmount()
	}

	if len(m.Fee.ReceiverString) > 0 {
		var enc *jsonenc.Encoder
		if err := config.LoadJSONEncoderContextValue(ctx, &enc); err != nil {
			return ctx, err
		}

		if address, err := base.DecodeAddressFromString(enc, m.Fee.ReceiverString); err != nil {
			return ctx, err
		} else {
			m.Fee.Receiver = address
		}
	}

	cmd.Log().Debug().Interface("fee_amount", json.RawMessage([]byte(m.Fee.FeeAmount.Verbose()))).Msg("fee amount loaded")

	return context.WithValue(ctx, ContextValueFeeDesign, m.Fee), nil
}

func (cmd *BaseNodeCommand) hookValidateFeeConfig(ctx context.Context) (context.Context, error) {
	var design FeeDesign
	switch err := LoadFeeDesignContextValue(ctx, &design); {
	case err != nil:
		return ctx, err
	case design.FeeAmount == nil:
		return ctx, xerrors.Errorf("empty fee amount")
	}

	if _, ok := design.FeeAmount.(currency.NilFeeAmount); ok {
		cmd.Log().Debug().Msg("nil fee")
	}

	if f, err := cmd.checkFeeReceiver(ctx, design.Receiver); err != nil {
		return ctx, err
	} else {
		design.ReceiverFunc = f
	}

	cmd.Log().Debug().
		Interface("fee_amount", json.RawMessage([]byte(design.FeeAmount.Verbose()))).
		Msg("fee amount validated")

	return context.WithValue(ctx, ContextValueFeeDesign, design), nil
}

func (cmd *BaseNodeCommand) checkFeeReceiver(
	ctx context.Context,
	receiver base.Address,
) (func() (base.Address, error), error) {
	var enc *jsonenc.Encoder
	if err := config.LoadJSONEncoderContextValue(ctx, &enc); err != nil {
		return nil, err
	}

	var st storage.Storage
	if err := process.LoadStorageContextValue(ctx, &st); err != nil {
		return nil, err
	}

	var blockFS *storage.BlockFS
	if err := process.LoadBlockFSContextValue(ctx, &blockFS); err != nil {
		return nil, err
	}

	if receiver != nil {
		switch _, found, err := st.State(currency.StateKeyAccount(receiver)); {
		case err != nil:
			return nil, xerrors.Errorf("failed to find fee receiver, %v: %w", receiver, err)
		case !found:
			return nil, xerrors.Errorf("fee receiver, %v does not exist", receiver)
		}
	} else if gac, _, exists, err := cmd.loadGenesisAccount(enc, st, blockFS); err != nil {
		return nil, err
	} else if exists {
		receiver = gac.Address()
	}

	if receiver != nil {
		return func() (base.Address, error) {
			return receiver, nil
		}, nil
	}

	return func() (base.Address, error) {
		if receiver != nil {
			return receiver, nil
		}

		switch gac, _, exists, err := cmd.loadGenesisAccount(enc, st, blockFS); {
		case err != nil:
			return nil, err
		case exists:
			receiver = gac.Address()

			return receiver, nil
		default:
			return nil, nil
		}
	}, nil
}

func initializeProposalProcessor(dp isaac.ProposalProcessor, opr isaac.OperationProcessor) error {
	for _, hinter := range []hint.Hinter{
		currency.CreateAccounts{},
		currency.KeyUpdater{},
		currency.Transfers{},
	} {
		if _, err := dp.AddOperationProcessor(hinter, opr); err != nil {
			return err
		}
	}

	return nil
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

	var fee FeeDesign
	if err := LoadFeeDesignContextValue(ctx, &fee); err != nil {
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

	m["fee"] = fee
	m["digest"] = dd

	log.Debug().Interface("config", m).Msg("config loaded")

	return ctx, nil
}
