package cmds

import (
	"context"

	"github.com/spikeekips/mitum-currency/digest"
	"github.com/spikeekips/mitum/base"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/launch/pm"
)

type CleanByHeightStorageCommand struct {
	*mitumcmds.CleanByHeightStorageCommand
	*BaseNodeCommand
}

func newCleanByHeightStorageCommand() (CleanByHeightStorageCommand, error) {
	co := mitumcmds.NewCleanByHeightStorageCommand()
	cmd := CleanByHeightStorageCommand{
		CleanByHeightStorageCommand: &co,
		BaseNodeCommand:             NewBaseNodeCommand(co.Logging),
	}

	hooks := []pm.Hook{
		pm.NewHook(pm.HookPrefixPost, ProcessNameDigestDatabase,
			"set_digest_clean_storage_by_height", func(ctx context.Context) (context.Context, error) {
				var st *digest.Database
				if err := LoadDigestDatabaseContextValue(ctx, &st); err != nil {
					return ctx, err
				}

				return context.WithValue(ctx, mitumcmds.ContextValueCleanDatabaseByHeight,
					func(ctx context.Context, h base.Height) error {
						return st.CleanByHeight(ctx, h)
					}), nil
			}),
	}

	ps, err := cmd.BaseProcesses(co.Processes())
	if err != nil {
		return cmd, err
	}

	processes := []pm.Process{
		ProcessorDigestDatabase,
	}

	for i := range processes {
		if err := ps.AddProcess(processes[i], false); err != nil {
			return CleanByHeightStorageCommand{}, err
		}
	}

	for i := range hooks {
		if err := hooks[i].Add(ps); err != nil {
			return CleanByHeightStorageCommand{}, err
		}
	}

	_ = cmd.SetProcesses(ps)

	return cmd, nil
}
