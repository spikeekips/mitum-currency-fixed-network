package digest

import (
	"github.com/spikeekips/mitum/network"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type NodeInfoPackerJSON struct {
	jsonenc.HintedHead
	network.NodeInfoV0PackerJSON
}

func (ni NodeInfo) MarshalJSON() ([]byte, error) {
	pj := NodeInfoPackerJSON{
		HintedHead:           jsonenc.NewHintedHead(ni.Hint()),
		NodeInfoV0PackerJSON: ni.NodeInfoV0.JSONPacker(),
	}

	return jsonenc.Marshal(pj)
}

func (ni *NodeInfo) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	internal := new(network.NodeInfoV0)
	if err := internal.UnpackJSON(b, enc); err != nil {
		return err
	}

	ni.NodeInfoV0 = *internal

	return nil
}
