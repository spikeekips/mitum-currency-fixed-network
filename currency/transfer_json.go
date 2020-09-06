package currency

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type TransferItemJSONPacker struct {
	RC base.Address `json:"receiver"`
	AM Amount       `json:"amount"`
}

func (tff TransferItem) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(TransferItemJSONPacker{
		RC: tff.receiver,
		AM: tff.amount,
	})
}

type TransferItemJSONUnpacker struct {
	RC base.AddressDecoder `json:"receiver"`
	AM Amount              `json:"amount"`
}

func (tff *TransferItem) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var utff TransferItemJSONUnpacker
	if err := jsonenc.Unmarshal(b, &utff); err != nil {
		return err
	}

	return tff.unpack(enc, utff.RC, utff.AM)
}

type TransferFactJSONPacker struct {
	jsonenc.HintedHead
	H  valuehash.Hash `json:"hash"`
	TK []byte         `json:"token"`
	SD base.Address   `json:"sender"`
	IT []TransferItem `json:"items"`
}

func (tff TransfersFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(TransferFactJSONPacker{
		HintedHead: jsonenc.NewHintedHead(tff.Hint()),
		H:          tff.h,
		TK:         tff.token,
		SD:         tff.sender,
		IT:         tff.items,
	})
}

type TransferFactJSONUnpacker struct {
	H  valuehash.Bytes     `json:"hash"`
	TK []byte              `json:"token"`
	SD base.AddressDecoder `json:"sender"`
	IT []json.RawMessage   `json:"items"`
}

func (tff *TransfersFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var utff TransferFactJSONUnpacker
	if err := jsonenc.Unmarshal(b, &utff); err != nil {
		return err
	}

	its := make([]TransferItem, len(utff.IT))
	for i := range utff.IT {
		it := new(TransferItem)
		if err := it.UnpackJSON(utff.IT[i], enc); err != nil {
			return err
		}

		its[i] = *it
	}

	return tff.unpack(enc, utff.H, utff.TK, utff.SD, its)
}

type TransferJSONPacker struct {
	jsonenc.HintedHead
	FC base.Fact            `json:"fact"`
	H  valuehash.Hash       `json:"hash"`
	FS []operation.FactSign `json:"fact_signs"`
}

func (tf Transfers) MarshalJSON() ([]byte, error) {
	m := tf.BaseOperation.JSONM()
	m["memo"] = tf.Memo

	return jsonenc.Marshal(m)
}

func (tf *Transfers) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ubo operation.BaseOperation
	if err := ubo.UnpackJSON(b, enc); err != nil {
		return err
	}

	*tf = Transfers{BaseOperation: ubo}

	var um MemoJSONUnpacker
	if err := enc.Unmarshal(b, &um); err != nil {
		return err
	} else {
		tf.Memo = um.Memo
	}

	return nil
}
