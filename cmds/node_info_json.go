package cmds

import (
	"encoding/json"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/network"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"golang.org/x/xerrors"
)

type NodeInfoGenesisPackerJSON struct {
	GA currency.Account `json:"account"`
	GB currency.Amount  `json:"balance"`
}

type NodeInfoPackerJSON struct {
	jsonenc.HintedHead
	network.NodeInfoV0PackerJSON
	FA json.RawMessage            `json:"fee_amount"`
	GE *NodeInfoGenesisPackerJSON `json:"genesis,omitempty"`
}

func (ni NodeInfo) MarshalJSON() ([]byte, error) {
	pj := NodeInfoPackerJSON{
		HintedHead:           jsonenc.NewHintedHead(ni.Hint()),
		NodeInfoV0PackerJSON: ni.NodeInfoV0.JSONPacker(),
		FA:                   json.RawMessage([]byte(ni.feeAmount)),
	}

	if ni.genesisBalance.Compare(currency.ZeroAmount) < 1 {
		pj.GE = nil
	} else {
		pj.GE = &NodeInfoGenesisPackerJSON{
			GA: ni.genesisAccount,
			GB: ni.genesisBalance,
		}
	}

	return jsonenc.Marshal(pj)
}

type NodeInfoGenesisUnpackerJSON struct {
	GA json.RawMessage `json:"account"`
	GB currency.Amount `json:"balance"`
}

type NodeInfoUnpackerJSON struct {
	FA json.RawMessage `json:"fee_amount"`
	GE json.RawMessage `json:"genesis"`
}

func (ni *NodeInfo) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	internal := new(network.NodeInfoV0)
	if err := internal.UnpackJSON(b, enc); err != nil {
		return err
	}

	var nni NodeInfoUnpackerJSON
	if err := enc.Unmarshal(b, &nni); err != nil {
		return err
	}

	if len(nni.GE) > 0 {
		var ge NodeInfoGenesisUnpackerJSON
		if err := enc.Unmarshal(nni.GE, &ge); err != nil {
			return err
		} else {
			var ga currency.Account
			if hinter, err := enc.DecodeByHint(ge.GA); err != nil {
				return err
			} else if i, ok := hinter.(currency.Account); !ok {
				return xerrors.Errorf("not Account in NodeInfo, %T", hinter)
			} else {
				ga = i
			}

			ni.genesisAccount = ga
			ni.genesisBalance = ge.GB
		}
	}

	ni.NodeInfoV0 = *internal
	ni.feeAmount = string(nni.FA)

	return nil
}
