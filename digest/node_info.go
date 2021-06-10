package digest

import (
	"github.com/spikeekips/mitum/network"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
)

var (
	NodeInfoType = hint.Type("mitum-currency-node-info")
	NodeInfoHint = hint.NewHint(NodeInfoType, "v0.0.1")
)

type NodeInfo struct {
	network.NodeInfoV0
}

func NewNodeInfo(ni network.NodeInfoV0) NodeInfo {
	return NodeInfo{
		NodeInfoV0: ni,
	}
}

func (NodeInfo) Hint() hint.Hint {
	return NodeInfoHint
}

func (ni NodeInfo) String() string {
	return jsonenc.ToString(ni)
}
