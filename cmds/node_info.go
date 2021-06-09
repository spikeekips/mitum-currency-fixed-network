package cmds

import (
	"golang.org/x/xerrors"

	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"

	"github.com/spikeekips/mitum-currency/digest"
)

type NodeInfoCommand struct {
	*mitumcmds.NodeInfoCommand
}

func NewNodeInfoCommand() NodeInfoCommand {
	var co *mitumcmds.NodeInfoCommand
	{
		i := mitumcmds.NewNodeInfoCommand()
		co = &i
	}
	cmd := NodeInfoCommand{
		NodeInfoCommand: co,
	}

	return cmd
}

func (cmd *NodeInfoCommand) Run(version util.Version) error {
	if cmd.Encoders() == nil {
		if _, err := cmd.LoadEncoders(Types, Hinters); err != nil {
			return err
		}
	}

	return cmd.NodeInfoCommand.Run(version)
}

func NodeInfoHandler(
	handler network.NodeInfoHandler,
) network.NodeInfoHandler {
	return func() (network.NodeInfo, error) {
		var ni network.NodeInfoV0
		if i, err := handler(); err != nil {
			return nil, err
		} else if j, ok := i.(network.NodeInfoV0); !ok {
			return nil, xerrors.Errorf("unsupported NodeInfo, %T", i)
		} else {
			ni = j
		}

		return digest.NewNodeInfo(ni), nil
	}
}
