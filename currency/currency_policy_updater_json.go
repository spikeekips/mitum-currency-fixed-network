package currency

import (
	"encoding/json"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type CurrencyPolicyUpdaterFactJSONPacker struct {
	jsonenc.HintedHead
	H  valuehash.Hash `json:"hash"`
	TK []byte         `json:"token"`
	CI CurrencyID     `json:"currency"`
	PO CurrencyPolicy `json:"policy"`
}

func (fact CurrencyPolicyUpdaterFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(CurrencyPolicyUpdaterFactJSONPacker{
		HintedHead: jsonenc.NewHintedHead(fact.Hint()),
		H:          fact.h,
		TK:         fact.token,
		CI:         fact.cid,
		PO:         fact.policy,
	})
}

type CurrencyPolicyUpdaterFactJSONUnpacker struct {
	H  valuehash.Bytes `json:"hash"`
	TK []byte          `json:"token"`
	CI string          `json:"currency"`
	PO json.RawMessage `json:"policy"`
}

func (fact *CurrencyPolicyUpdaterFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ufact CurrencyPolicyUpdaterFactJSONUnpacker
	if err := jsonenc.Unmarshal(b, &ufact); err != nil {
		return err
	}

	return fact.unpack(enc, ufact.H, ufact.TK, ufact.CI, ufact.PO)
}

func (op *CurrencyPolicyUpdater) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ubo BaseOperation
	if err := ubo.UnpackJSON(b, enc); err != nil {
		return err
	}

	op.BaseOperation = ubo

	return nil
}
