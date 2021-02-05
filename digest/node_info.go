package digest

import (
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
}

func NewNodeInfo(ni network.NodeInfoV0) NodeInfo {
	return NodeInfo{
		NodeInfoV0: ni,
	}
}

func (ni NodeInfo) Hint() hint.Hint {
	return NodeInfoHint
}

func (ni NodeInfo) String() string {
	return jsonenc.ToString(ni)
}
