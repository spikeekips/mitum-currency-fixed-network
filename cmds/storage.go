package cmds

import (
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
)

type StorageCommand struct {
	Download                    mitumcmds.BlockDownloadCommand    `cmd:"" name:"download" help:"download block data"`
	BlockDataVerify             mitumcmds.BlockDataVerifyCommand  `cmd:"" name:"verify-blockdata" help:"verify block data"` // revive:disable-line:line-length-limit
	DatabaseVerify              mitumcmds.DatabaseVerifyCommand   `cmd:"" name:"verify-database" help:"verify database"`    // revive:disable-line:line-length-limit
	CleanStorage                mitumcmds.CleanStorageCommand     `cmd:"" name:"clean" help:"clean storage"`
	CleanByHeightStorageCommand CleanByHeightStorageCommand       `cmd:"" name:"clean-by-height" help:"clean storage by height"` // revive:disable-line:line-length-limit
	Restore                     mitumcmds.RestoreCommand          `cmd:"" help:"restore"`
	SetBlockDataMaps            mitumcmds.SetBlockDataMapsCommand `cmd:"" name:"set-blockdatamaps" help:"set blockdatamaps"` // revive:disable-line:line-length-limit
}

func NewStorageCommand() StorageCommand {
	return StorageCommand{
		Download:                    mitumcmds.NewBlockDownloadCommand(Types, Hinters),
		BlockDataVerify:             mitumcmds.NewBlockDataVerifyCommand(Types, Hinters),
		DatabaseVerify:              mitumcmds.NewDatabaseVerifyCommand(Types, Hinters),
		CleanStorage:                mitumcmds.NewCleanStorageCommand(false),
		CleanByHeightStorageCommand: NewCleanByHeightStorageCommand(),
		Restore:                     mitumcmds.NewRestoreCommand(Types, Hinters),
		SetBlockDataMaps:            mitumcmds.NewSetBlockDataMapsCommand(),
	}
}
