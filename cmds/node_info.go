package cmds

import (
	"net/url"

	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type NodeInfoCommand struct {
	BaseCommand
	URL    *url.URL `arg:"" name:"node url" help:"remote mitum url (default: ${node_url})" required:"" default:"${node_url}"` // nolint
	Pretty bool     `name:"pretty" help:"pretty format"`
}

func (cmd *NodeInfoCommand) Run(flags *MainFlags, version util.Version, log logging.Logger) error {
	_ = cmd.BaseCommand.Run(flags, version, log)

	var channel network.Channel
	if ch, err := launcher.LoadNodeChannel(cmd.URL, encs); err != nil {
		return err
	} else {
		channel = ch
	}

	if n, err := channel.NodeInfo(); err != nil {
		return err
	} else {
		cmd.pretty(cmd.Pretty, n)
	}

	return nil
}
