package mc

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type TransferFactJSONPacker struct {
	jsonenc.HintedHead
	H  valuehash.Hash `json:"hash"`
	TK []byte         `json:"token"`
	SD Address        `json:"sender"`
	RC Address        `json:"receiver"`
	AM Amount         `json:"amount"`
}

func (tff TransferFact) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(TransferFactJSONPacker{
		HintedHead: jsonenc.NewHintedHead(tff.Hint()),
		H:          tff.h,
		TK:         tff.token,
		SD:         tff.sender,
		RC:         tff.receiver,
		AM:         tff.amount,
	})
}

type TransferFactJSONUnpacker struct {
	H  valuehash.Bytes `json:"hash"`
	TK []byte          `json:"token"`
	SD Address         `json:"sender"`
	RC Address         `json:"receiver"`
	AM Amount          `json:"amount"`
}

func (tff *TransferFact) UnmarshalJSON(b []byte) error {
	var utff TransferFactJSONUnpacker
	if err := util.JSON.Unmarshal(b, &utff); err != nil {
		return err
	}

	tff.h = utff.H
	tff.token = utff.TK
	tff.sender = utff.SD
	tff.receiver = utff.RC
	tff.amount = utff.AM

	return nil
}

type TransferJSONPacker struct {
	jsonenc.HintedHead
	FC base.Fact            `json:"fact"`
	H  valuehash.Hash       `json:"hash"`
	FS []operation.FactSign `json:"fact_signs"`
}

func (tf Transfer) MarshalJSON() ([]byte, error) {
	m := tf.BaseOperation.JSONM()
	m["memo"] = tf.Memo

	return util.JSON.Marshal(m)
}

func (tf *Transfer) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ubo operation.BaseOperation
	if err := ubo.UnpackJSON(b, enc); err != nil {
		return err
	}

	*tf = Transfer{BaseOperation: ubo}

	var um MemoJSONUnpacker
	if err := enc.Unmarshal(b, &um); err != nil {
		return err
	} else {
		tf.Memo = um.Memo
	}

	return nil
}
