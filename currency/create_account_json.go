package currency

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type CreateAccountItemJSONPacker struct {
	KS Keys   `json:"keys"`
	AM Amount `json:"amount"`
}

func (cai CreateAccountItem) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(CreateAccountItemJSONPacker{
		KS: cai.keys,
		AM: cai.amount,
	})
}

type CreateAccountItemJSONUnpacker struct {
	KS json.RawMessage `json:"keys"`
	AM Amount          `json:"amount"`
}

func (cai *CreateAccountItem) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uca CreateAccountItemJSONUnpacker
	if err := util.JSON.Unmarshal(b, &uca); err != nil {
		return err
	}

	return cai.unpack(enc, uca.KS, uca.AM)
}

type CreateAccountFactJSONPacker struct {
	jsonenc.HintedHead
	H  valuehash.Hash      `json:"hash"`
	TK []byte              `json:"token"`
	SD base.Address        `json:"sender"`
	IT []CreateAccountItem `json:"items"`
}

func (caf CreateAccountsFact) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(CreateAccountFactJSONPacker{
		HintedHead: jsonenc.NewHintedHead(caf.Hint()),
		H:          caf.h,
		TK:         caf.token,
		SD:         caf.sender,
		IT:         caf.items,
	})
}

type CreateAccountFactJSONUnpacker struct {
	H  valuehash.Bytes     `json:"hash"`
	TK []byte              `json:"token"`
	SD base.AddressDecoder `json:"sender"`
	IT []json.RawMessage   `json:"items"`
}

func (caf *CreateAccountsFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uca CreateAccountFactJSONUnpacker
	if err := util.JSON.Unmarshal(b, &uca); err != nil {
		return err
	}

	its := make([]CreateAccountItem, len(uca.IT))
	for i := range uca.IT {
		it := new(CreateAccountItem)
		if err := it.UnpackJSON(uca.IT[i], enc); err != nil {
			return err
		}

		its[i] = *it
	}

	return caf.unpack(enc, uca.H, uca.TK, uca.SD, its)
}

func (ca CreateAccounts) MarshalJSON() ([]byte, error) {
	m := ca.BaseOperation.JSONM()
	m["memo"] = ca.Memo

	return util.JSON.Marshal(m)
}

func (ca *CreateAccounts) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ubo operation.BaseOperation
	if err := ubo.UnpackJSON(b, enc); err != nil {
		return err
	}

	*ca = CreateAccounts{BaseOperation: ubo}

	var um MemoJSONUnpacker
	if err := enc.Unmarshal(b, &um); err != nil {
		return err
	} else {
		ca.Memo = um.Memo
	}

	return nil
}
