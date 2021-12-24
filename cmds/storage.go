package cmds

import (
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
)

type StorageCommand struct {
	Download                    mitumcmds.BlockDownloadCommand    `cmd:"" name:"download" help:"download block data"`
	BlockdataVerify             mitumcmds.BlockdataVerifyCommand  `cmd:"" name:"verify-blockdata" help:"verify block data"` // revive:disable-line:line-length-limit
	DatabaseVerify              mitumcmds.DatabaseVerifyCommand   `cmd:"" name:"verify-database" help:"verify database"`    // revive:disable-line:line-length-limit
	CleanStorage                CleanStorageCommand               `cmd:"" name:"clean" help:"clean storage"`
	CleanByHeightStorageCommand CleanByHeightStorageCommand       `cmd:"" name:"clean-by-height" help:"clean storage by height"` // revive:disable-line:line-length-limit
	Restore                     restoreCommand                    `cmd:"" help:"restore blocks from blockdata"`
	SetBlockdataMaps            mitumcmds.SetBlockdataMapsCommand `cmd:"" name:"set-blockdatamaps" help:"set blockdatamaps"` // revive:disable-line:line-length-limit
}

func NewStorageCommand() (StorageCommand, error) {
	restoreCommand, err := newRestoreCommand()
	if err != nil {
		return StorageCommand{}, err
	}

	cleanStorageCommand, err := newCleanStorageCommand(false)
	if err != nil {
		return StorageCommand{}, err
	}

	cleanByHeightStorageCommand, err := newCleanByHeightStorageCommand()
	if err != nil {
		return StorageCommand{}, err
	}

	return StorageCommand{
		Download:                    mitumcmds.NewBlockDownloadCommand(Types, Hinters),
		BlockdataVerify:             mitumcmds.NewBlockdataVerifyCommand(Types, Hinters),
		DatabaseVerify:              mitumcmds.NewDatabaseVerifyCommand(Types, Hinters),
		CleanStorage:                cleanStorageCommand,
		CleanByHeightStorageCommand: cleanByHeightStorageCommand,
		Restore:                     restoreCommand,
		SetBlockdataMaps:            mitumcmds.NewSetBlockdataMapsCommand(),
	}, nil
}
