package cmds

import (
	"context"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/isaac"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/storage"
	"golang.org/x/xerrors"
)

var (
	GenesisAccountKey = "genesis_account"
	GenesisBalanceKey = "genesis_balance"
)

var InitCommandHooks = func(cmd *InitCommand) []pm.Hook {
	return []pm.Hook{
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameProposalProcessor,
			"apply_fee", cmd.hookApplyNilFee).SetOverride(true),
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameGenerateGenesisBlock,
			"save_genesis_info", cmd.hookSaveGenesisInfo).SetOverride(true),
	}
}

type InitCommand struct {
	*BaseNodeCommand
	*mitumcmds.InitCommand
}

func NewInitCommand(dryrun bool) (InitCommand, error) {
	co := mitumcmds.NewInitCommand(dryrun)
	cmd := InitCommand{
		InitCommand:     &co,
		BaseNodeCommand: NewBaseNodeCommand(co.Logging),
	}

	ps := co.Processes()
	if i, err := cmd.BaseProcesses(ps); err != nil {
		return cmd, err
	} else {
		ps = i
	}

	hooks := InitCommandHooks(&cmd)
	for i := range hooks {
		if err := hooks[i].Add(ps); err != nil {
			return cmd, err
		}
	}

	_ = cmd.SetProcesses(ps)

	return cmd, nil
}

func (cmd *InitCommand) hookApplyNilFee(ctx context.Context) (context.Context, error) {
	var proposalProcessor isaac.ProposalProcessor
	if err := process.LoadProposalProcessorContextValue(ctx, &proposalProcessor); err != nil {
		return ctx, err
	}

	// NOTE NilFeeAmount will be applied whatever design defined
	if err := initializeProposalProcessor(
		proposalProcessor,
		currency.NewOperationProcessor(currency.NewNilFeeAmount(), nil),
	); err != nil {
		return ctx, err
	}

	cmd.Log().Debug().Msg("nil fee amount applied for init")

	return context.WithValue(ctx, process.ContextValueProposalProcessor, proposalProcessor), nil
}

func (cmd *InitCommand) hookSaveGenesisInfo(ctx context.Context) (context.Context, error) {
	var st storage.Storage
	if err := process.LoadStorageContextValue(ctx, &st); err != nil {
		return ctx, err
	}

	var genesis block.Block
	if err := process.LoadGenesisBlockContextValue(ctx, &genesis); err != nil {
		return ctx, err
	}

	if _, _, err := cmd.saveGenesisAccountInfo(st, genesis); err != nil {
		return ctx, xerrors.Errorf("failed to save genesis account for init: %w", err)
	} else {
		return ctx, nil
	}
}
