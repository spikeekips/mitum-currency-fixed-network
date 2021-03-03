package cmds

import (
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
)

type NodeCommand struct {
	Init         InitCommand                   `cmd:"" help:"initialize node"`
	Run          RunCommand                    `cmd:"" help:"run node"`
	Info         NodeInfoCommand               `cmd:"" help:"node information"`
	CleanStorage mitumcmds.CleanStorageCommand `cmd:"" help:"clean storage"`
}

func NewNodeCommand() (NodeCommand, error) {
	var initCommand InitCommand
	if i, err := NewInitCommand(false); err != nil {
		return NodeCommand{}, err
	} else {
		initCommand = i
	}

	var runCommand RunCommand
	if i, err := NewRunCommand(false); err != nil {
		return NodeCommand{}, err
	} else {
		runCommand = i
	}

	return NodeCommand{
		Init:         initCommand,
		Run:          runCommand,
		Info:         NewNodeInfoCommand(),
		CleanStorage: mitumcmds.NewCleanStorageCommand(false),
	}, nil
}
