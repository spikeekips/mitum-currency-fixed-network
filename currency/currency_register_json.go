package currency

import (
	"encoding/json"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type CurrencyRegisterFactJSONPacker struct {
	jsonenc.HintedHead
	H  valuehash.Hash `json:"hash"`
	TK []byte         `json:"token"`
	CR CurrencyDesign `json:"currency"`
}

func (fact CurrencyRegisterFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(CurrencyRegisterFactJSONPacker{
		HintedHead: jsonenc.NewHintedHead(fact.Hint()),
		H:          fact.h,
		TK:         fact.token,
		CR:         fact.currency,
	})
}

type CurrencyRegisterFactJSONUnpacker struct {
	H  valuehash.Bytes `json:"hash"`
	TK []byte          `json:"token"`
	CR json.RawMessage `json:"currency"`
}

func (fact *CurrencyRegisterFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ufact CurrencyRegisterFactJSONUnpacker
	if err := jsonenc.Unmarshal(b, &ufact); err != nil {
		return err
	}

	return fact.unpack(enc, ufact.H, ufact.TK, ufact.CR)
}

func (op *CurrencyRegister) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ubo BaseOperation
	if err := ubo.UnpackJSON(b, enc); err != nil {
		return err
	}

	op.BaseOperation = ubo

	return nil
}
