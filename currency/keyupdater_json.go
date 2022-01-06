package currency // nolint: dupl

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type KeyUpdaterFactJSONPacker struct {
	jsonenc.HintedHead
	H  valuehash.Hash `json:"hash"`
	TK []byte         `json:"token"`
	TG base.Address   `json:"target"`
	KS AccountKeys    `json:"keys"`
	CR CurrencyID     `json:"currency"`
}

func (fact KeyUpdaterFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(KeyUpdaterFactJSONPacker{
		HintedHead: jsonenc.NewHintedHead(fact.Hint()),
		H:          fact.h,
		TK:         fact.token,
		TG:         fact.target,
		KS:         fact.keys,
		CR:         fact.currency,
	})
}

type KeyUpdaterFactJSONUnpacker struct {
	H  valuehash.Bytes     `json:"hash"`
	TK []byte              `json:"token"`
	TG base.AddressDecoder `json:"target"`
	KS json.RawMessage     `json:"keys"`
	CR string              `json:"currency"`
}

func (fact *KeyUpdaterFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ufact KeyUpdaterFactJSONUnpacker
	if err := enc.Unmarshal(b, &ufact); err != nil {
		return err
	}

	return fact.unpack(enc, ufact.H, ufact.TK, ufact.TG, ufact.KS, ufact.CR)
}

func (op *KeyUpdater) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ubo BaseOperation
	if err := ubo.UnpackJSON(b, enc); err != nil {
		return err
	}

	op.BaseOperation = ubo

	return nil
}
