package cmds

import (
	"net/url"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
)

var (
	NodeInfoType = hint.MustNewType(0xa0, 0x15, "mitum-currency-node-info")
	NodeInfoHint = hint.MustHint(NodeInfoType, "0.0.1")
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

type NodeInfo struct {
	network.NodeInfoV0
	feeAmount      string
	genesisAccount currency.Account
	genesisBalance currency.Amount
}

func NewNodeInfo(
	ni network.NodeInfoV0,
	fa currency.FeeAmount,
	genesisAccount currency.Account,
	genesisBalance currency.Amount,
) NodeInfo {
	if fa == nil {
		fa = currency.NewNilFeeAmount()
	}

	return NodeInfo{
		NodeInfoV0:     ni,
		feeAmount:      fa.Verbose(),
		genesisAccount: genesisAccount,
		genesisBalance: genesisBalance,
	}
}

func (ni NodeInfo) Hint() hint.Hint {
	return NodeInfoHint
}

func (ni NodeInfo) String() string {
	return jsonenc.ToString(ni)
}

func (ni NodeInfo) FeeAmount() string {
	return ni.feeAmount
}

func (ni NodeInfo) GenesisAccount() currency.Account {
	return ni.genesisAccount
}

func (ni NodeInfo) GenesisBalance() currency.Amount {
	return ni.genesisBalance
}
