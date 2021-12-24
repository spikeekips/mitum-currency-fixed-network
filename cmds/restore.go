package cmds

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum-currency/digest"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/util"
)

var restoreCommandHooks = func(cmd *restoreCommand) []pm.Hook {
	genesisOperationHandlers := map[string]process.HookHandlerGenesisOperations{
		"genesis-currencies": nil,
	}

	for k, v := range process.DefaultHookHandlersGenesisOperations {
		genesisOperationHandlers[k] = v
	}

	return []pm.Hook{
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameConfig,
			process.HookNameConfigGenesisOperations, process.HookGenesisOperationFunc(genesisOperationHandlers)).
			SetOverride(true),
		pm.NewHook(pm.HookPrefixPost, ProcessNameDigestDatabase,
			"set_digest_when_block_saved", func(ctx context.Context) (context.Context, error) {
				var st *digest.Database
				if err := LoadDigestDatabaseContextValue(ctx, &st); err != nil {
					if errors.Is(err, util.ContextValueNotFoundError) {
						return ctx, nil
					}

					return ctx, err
				}

				ctx = context.WithValue(ctx, mitumcmds.ContextValueWhenBlockSaved, func(blk block.Block) error {
					return digest.DigestBlock(context.Background(), st, blk)
				})

				ctx = context.WithValue(ctx, mitumcmds.ContextValueWhenFinished, func(to base.Height) error {
					return st.SetLastBlock(to)
				})

				ctx = context.WithValue(ctx, mitumcmds.ContextValueCleanDatabase, func() error {
					return st.Clean()
				})

				ctx = context.WithValue(ctx, mitumcmds.ContextValueCleanDatabaseByHeight,
					func(ctx context.Context, h base.Height) error {
						return st.CleanByHeight(ctx, h)
					})

				return ctx, nil
			}).
			SetOverride(true),
	}
}

type restoreCommand struct {
	*mitumcmds.RestoreCommand
	*BaseNodeCommand
}

func newRestoreCommand() (restoreCommand, error) {
	co := mitumcmds.NewRestoreCommand()
	cmd := restoreCommand{
		RestoreCommand:  &co,
		BaseNodeCommand: NewBaseNodeCommand(co.Logging),
	}

	ps, err := cmd.BaseProcesses(co.Processes())
	if err != nil {
		return cmd, err
	}

	restoreCommandProcesses := []pm.Process{
		ProcessorDigestDatabase,
	}

	for i := range restoreCommandProcesses {
		if err := ps.AddProcess(restoreCommandProcesses[i], true); err != nil {
			return cmd, err
		}
	}

	hooks := restoreCommandHooks(&cmd)
	for i := range hooks {
		if err := hooks[i].Add(ps); err != nil {
			return cmd, err
		}
	}

	_ = cmd.SetProcesses(ps)

	return cmd, nil
}
