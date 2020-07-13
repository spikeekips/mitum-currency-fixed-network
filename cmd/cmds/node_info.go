package cmds

import (
	"fmt"
	"net/url"
	"os"

	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/network"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type NodeInfoCommand struct {
	URL *url.URL `arg:"" name:"node url" help:"remote mitum url (default: ${node_url})" required:"" default:"${node_url}"`
}

func (cmd *NodeInfoCommand) Run() error {
	var channel network.NetworkChannel
	if ch, err := launcher.LoadNodeChannel(cmd.URL, encs); err != nil {
		return err
	} else {
		channel = ch
	}

	if n, err := channel.NodeInfo(); err != nil {
		return err
	} else {
		_, _ = fmt.Fprintln(os.Stdout, jsonenc.ToString(n))
	}

	return nil
}
