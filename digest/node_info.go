package digest

import (
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/network"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
)

var (
	NodeInfoType = hint.MustNewType(0xa0, 0x15, "mitum-currency-node-info")
	NodeInfoHint = hint.MustHint(NodeInfoType, "0.0.1")
)

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
