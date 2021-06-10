package cmds

type NodeCommand struct {
	Init InitCommand     `cmd:"" help:"initialize node"`
	Run  RunCommand      `cmd:"" help:"run node"`
	Info NodeInfoCommand `cmd:"" help:"node information"`
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
		Init: initCommand,
		Run:  runCommand,
		Info: NewNodeInfoCommand(),
	}, nil
}
