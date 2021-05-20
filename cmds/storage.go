package cmds

import (
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
)

type StorageCommand struct {
	Download        mitumcmds.BlockDownloadCommand   `cmd:"" name:"download" help:"download block data"`
	BlockDataVerify mitumcmds.BlockDataVerifyCommand `cmd:"" name:"verify-blockdata" help:"verify block data"`
	DatabaseVerify  mitumcmds.DatabaseVerifyCommand  `cmd:"" name:"verify-database" help:"verify database"`
	CleanStorage    mitumcmds.CleanStorageCommand    `cmd:"" name:"clean" help:"clean storage"`
	Restore         mitumcmds.RestoreCommand         `cmd:"" help:"restore"`
}

func NewStorageCommand() StorageCommand {
	return StorageCommand{
		Download:        mitumcmds.NewBlockDownloadCommand(Hinters),
		BlockDataVerify: mitumcmds.NewBlockDataVerifyCommand(Hinters),
		DatabaseVerify:  mitumcmds.NewDatabaseVerifyCommand(Hinters),
		CleanStorage:    mitumcmds.NewCleanStorageCommand(false),
		Restore:         mitumcmds.NewRestoreCommand(Hinters),
	}
}
