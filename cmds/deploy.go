package cmds

import (
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
)

type DeployKeyCommand struct {
	New    mitumcmds.DeployKeyNewCommand    `cmd:"" name:"new" help:"request new deploy key"`
	Keys   mitumcmds.DeployKeyKeysCommand   `cmd:"" name:"keys" help:"deploy keys"`
	Key    mitumcmds.DeployKeyKeyCommand    `cmd:"" name:"key" help:"deploy key"`
	Revoke mitumcmds.DeployKeyRevokeCommand `cmd:"" name:"revoke" help:"revoke deploy key"`
}

func NewDeployKeyCommand() DeployKeyCommand {
	return DeployKeyCommand{
		New:    mitumcmds.NewDeployKeyNewCommand(),
		Keys:   mitumcmds.NewDeployKeyKeysCommand(),
		Key:    mitumcmds.NewDeployKeyKeyCommand(),
		Revoke: mitumcmds.NewDeployKeyRevokeCommand(),
	}
}

type DeployCommand struct {
	Key DeployKeyCommand `cmd:"" name:"key" help:"deploy key"`
}

func NewDeployCommand() DeployCommand {
	return DeployCommand{
		Key: NewDeployKeyCommand(),
	}
}
