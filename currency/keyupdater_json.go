package currency // nolint: dupl

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type KeyUpdaterFactJSONPacker struct {
	jsonenc.HintedHead
	H  valuehash.Hash `json:"hash"`
	TK []byte         `json:"token"`
	TG base.Address   `json:"target"`
	KS Keys           `json:"keys"`
}

func (ft KeyUpdaterFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(KeyUpdaterFactJSONPacker{
		HintedHead: jsonenc.NewHintedHead(ft.Hint()),
		H:          ft.h,
		TK:         ft.token,
		TG:         ft.target,
		KS:         ft.keys,
	})
}

type KeyUpdaterFactJSONUnpacker struct {
	H  valuehash.Bytes     `json:"hash"`
	TK []byte              `json:"token"`
	TG base.AddressDecoder `json:"target"`
	KS json.RawMessage     `json:"keys"`
}

func (ft *KeyUpdaterFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var utf KeyUpdaterFactJSONUnpacker
	if err := enc.Unmarshal(b, &utf); err != nil {
		return err
	}

	return ft.unpack(enc, utf.H, utf.TK, utf.TG, utf.KS)
}

func (op KeyUpdater) MarshalJSON() ([]byte, error) {
	m := op.BaseOperation.JSONM()
	m["memo"] = op.Memo

	return jsonenc.Marshal(m)
}

func (op *KeyUpdater) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ubo operation.BaseOperation
	if err := ubo.UnpackJSON(b, enc); err != nil {
		return err
	}

	*op = KeyUpdater{BaseOperation: ubo}

	var um MemoJSONUnpacker
	if err := enc.Unmarshal(b, &um); err != nil {
		return err
	} else {
		op.Memo = um.Memo
	}

	return nil
}
