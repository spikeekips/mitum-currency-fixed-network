package cmds

import (
	"context"

	"github.com/spikeekips/mitum-currency/digest"
	"github.com/spikeekips/mitum/base"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type CleanByHeightStorageCommand struct {
	*mitumcmds.CleanByHeightStorageCommand
	st *digest.Database
}

func NewCleanByHeightStorageCommand() CleanByHeightStorageCommand {
	co := mitumcmds.NewCleanByHeightStorageCommand()
	cmd := CleanByHeightStorageCommand{
		CleanByHeightStorageCommand: &co,
	}

	ps := cmd.Processes()

	hooks := []pm.Hook{
		pm.NewHook(pm.HookPrefixPre, process.ProcessNameLocalNode,
			"mitum-currency-check-clean-by-height-storage", cmd.check).
			SetDir(mitumcmds.HookNameCleanByHeightStorage, pm.HookDirBefore),
		pm.NewHook(pm.HookPrefixPre, process.ProcessNameLocalNode,
			"mitum-currency-clean-by-height-storage", cmd.clean).
			SetDir(mitumcmds.HookNameCleanByHeightStorage, pm.HookDirAfter),
	}

	for i := range hooks {
		hook := hooks[i]
		if err := hook.Add(ps); err != nil {
			panic(err)
		}
	}

	_ = cmd.SetProcesses(ps)

	return cmd
}

func (cmd *CleanByHeightStorageCommand) check(ctx context.Context) (context.Context, error) {
	var mst *mongodbstorage.Database
	if err := LoadDatabaseContextValue(ctx, &mst); err != nil {
		return ctx, err
	}

	st, err := loadDigestDatabase(mst, false)
	if err != nil {
		return ctx, err
	}

	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	_ = st.SetLogging(log)

	cmd.st = st

	cmd.Log().Debug().Msg("digest database found")

	return ctx, nil
}

func (cmd *CleanByHeightStorageCommand) clean(ctx context.Context) (context.Context, error) {
	var dryrun bool
	switch err := util.LoadFromContextValue(ctx, mitumcmds.ContextValueDryRun, &dryrun); {
	case err != nil:
		return ctx, err
	case dryrun:
		return ctx, nil
	}

	var height base.Height
	if err := util.LoadFromContextValue(ctx, mitumcmds.ContextValueHeight, &height); err != nil {
		return ctx, err
	}

	if err := cmd.st.CleanByHeight(context.Background(), height); err != nil {
		return ctx, err
	}

	cmd.Log().Debug().Msg("digest database cleaned by height")

	return ctx, nil
}
