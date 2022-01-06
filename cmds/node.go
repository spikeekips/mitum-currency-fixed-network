package cmds

import (
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
)

type NodeCommand struct {
	Init          InitCommand                    `cmd:"" help:"initialize node"`
	Run           RunCommand                     `cmd:"" help:"run node"`
	Info          NodeInfoCommand                `cmd:"" help:"node information"`
	StartHandover mitumcmds.StartHandoverCommand `cmd:"" name:"start-handover" help:"start handover"`
}

func NewNodeCommand() (NodeCommand, error) {
	initCommand, err := NewInitCommand(false)
	if err != nil {
		return NodeCommand{}, err
	}

	runCommand, err := NewRunCommand(false)
	if err != nil {
		return NodeCommand{}, err
	}

	return NodeCommand{
		Init:          initCommand,
		Run:           runCommand,
		Info:          NewNodeInfoCommand(),
		StartHandover: mitumcmds.NewStartHandoverCommand(),
	}, nil
}
