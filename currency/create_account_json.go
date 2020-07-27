package currency

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type CreateAccountFactJSONPacker struct {
	jsonenc.HintedHead
	H  valuehash.Hash `json:"hash"`
	TK []byte         `json:"token"`
	SD base.Address   `json:"sender"`
	KS Keys           `json:"keys"`
	AM Amount         `json:"amount"`
}

func (caf CreateAccountFact) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(CreateAccountFactJSONPacker{
		HintedHead: jsonenc.NewHintedHead(caf.Hint()),
		H:          caf.h,
		TK:         caf.token,
		SD:         caf.sender,
		KS:         caf.keys,
		AM:         caf.amount,
	})
}

type CreateAccountFactJSONUnpacker struct {
	H  valuehash.Bytes     `json:"hash"`
	TK []byte              `json:"token"`
	SD base.AddressDecoder `json:"sender"`
	KS json.RawMessage     `json:"keys"`
	AM Amount              `json:"amount"`
}

func (caf *CreateAccountFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uca CreateAccountFactJSONUnpacker
	if err := util.JSON.Unmarshal(b, &uca); err != nil {
		return err
	}

	return caf.unpack(enc, uca.H, uca.TK, uca.SD, uca.KS, uca.AM)
}

func (ca CreateAccount) MarshalJSON() ([]byte, error) {
	m := ca.BaseOperation.JSONM()
	m["memo"] = ca.Memo

	return util.JSON.Marshal(m)
}

func (ca *CreateAccount) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ubo operation.BaseOperation
	if err := ubo.UnpackJSON(b, enc); err != nil {
		return err
	}

	*ca = CreateAccount{BaseOperation: ubo}

	var um MemoJSONUnpacker
	if err := enc.Unmarshal(b, &um); err != nil {
		return err
	} else {
		ca.Memo = um.Memo
	}

	return nil
}
