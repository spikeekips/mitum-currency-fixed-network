package currency

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type TransferFactJSONPacker struct {
	jsonenc.HintedHead
	H  valuehash.Hash  `json:"hash"`
	TK []byte          `json:"token"`
	SD base.Address    `json:"sender"`
	IT []TransfersItem `json:"items"`
}

func (fact TransfersFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(TransferFactJSONPacker{
		HintedHead: jsonenc.NewHintedHead(fact.Hint()),
		H:          fact.h,
		TK:         fact.token,
		SD:         fact.sender,
		IT:         fact.items,
	})
}

func (fact *TransfersFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ufact struct {
		H  valuehash.Bytes     `json:"hash"`
		TK []byte              `json:"token"`
		SD base.AddressDecoder `json:"sender"`
		IT json.RawMessage     `json:"items"`
	}
	if err := jsonenc.Unmarshal(b, &ufact); err != nil {
		return err
	}

	return fact.unpack(enc, ufact.H, ufact.TK, ufact.SD, ufact.IT)
}

func (op *Transfers) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ubo BaseOperation
	if err := ubo.UnpackJSON(b, enc); err != nil {
		return err
	}

	op.BaseOperation = ubo

	return nil
}
