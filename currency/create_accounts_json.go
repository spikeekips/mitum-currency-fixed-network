package currency

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type CreateAccountsFactJSONPacker struct {
	jsonenc.HintedHead
	H  valuehash.Hash       `json:"hash"`
	TK []byte               `json:"token"`
	SD base.Address         `json:"sender"`
	IT []CreateAccountsItem `json:"items"`
}

func (fact CreateAccountsFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(CreateAccountsFactJSONPacker{
		HintedHead: jsonenc.NewHintedHead(fact.Hint()),
		H:          fact.h,
		TK:         fact.token,
		SD:         fact.sender,
		IT:         fact.items,
	})
}

type CreateAccountsFactJSONUnpacker struct {
	H  valuehash.Bytes     `json:"hash"`
	TK []byte              `json:"token"`
	SD base.AddressDecoder `json:"sender"`
	IT json.RawMessage     `json:"items"`
}

func (fact *CreateAccountsFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uca CreateAccountsFactJSONUnpacker
	if err := jsonenc.Unmarshal(b, &uca); err != nil {
		return err
	}

	return fact.unpack(enc, uca.H, uca.TK, uca.SD, uca.IT)
}

func (op *CreateAccounts) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ubo BaseOperation
	if err := ubo.UnpackJSON(b, enc); err != nil {
		return err
	}

	op.BaseOperation = ubo

	return nil
}
