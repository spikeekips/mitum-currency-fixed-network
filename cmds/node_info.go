package cmds

import (
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum-currency/digest"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
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
		if _, err := cmd.LoadEncoders(Hinters); err != nil {
			return err
		}
	}

	return cmd.NodeInfoCommand.Run(version)
}

func NodeInfoHandler(
	handler network.NodeInfoHandler,
	fa currency.FeeAmount,
	fga func() *currency.Account,
	fgb func() *currency.Amount,
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

		var ga currency.Account
		if i := fga(); i == nil {
			return ni, nil
		} else {
			ga = *i
		}

		var gb currency.Amount
		if i := fgb(); i == nil {
			return ni, nil
		} else {
			gb = *i
		}

		return digest.NewNodeInfo(ni, fa, ga, gb), nil
	}
}
