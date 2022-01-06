package cmds

import (
	"context"

	"github.com/spikeekips/mitum-currency/digest"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/launch/pm"
)

type CleanStorageCommand struct {
	*mitumcmds.CleanStorageCommand
	*BaseNodeCommand
}

func newCleanStorageCommand(dryrun bool) (CleanStorageCommand, error) {
	co := mitumcmds.NewCleanStorageCommand(dryrun)
	cmd := CleanStorageCommand{
		CleanStorageCommand: &co,
		BaseNodeCommand:     NewBaseNodeCommand(co.Logging),
	}

	hooks := []pm.Hook{
		pm.NewHook(pm.HookPrefixPost, ProcessNameDigestDatabase,
			"set_digest_clean_storage", func(ctx context.Context) (context.Context, error) {
				var st *digest.Database
				if err := LoadDigestDatabaseContextValue(ctx, &st); err != nil {
					return ctx, err
				}

				return context.WithValue(ctx, mitumcmds.ContextValueCleanDatabase, func() error {
					return st.Clean()
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
			return CleanStorageCommand{}, err
		}
	}

	for i := range hooks {
		if err := hooks[i].Add(ps); err != nil {
			return CleanStorageCommand{}, err
		}
	}

	_ = cmd.SetProcesses(ps)

	return cmd, nil
}
