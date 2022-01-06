package currency

import (
	"github.com/spikeekips/mitum/base/operation"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func (op BaseOperation) MarshalJSON() ([]byte, error) {
	m := op.BaseOperation.JSONM()
	m["memo"] = op.Memo

	return jsonenc.Marshal(m)
}

func (op *BaseOperation) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ubo operation.BaseOperation
	if err := ubo.UnpackJSON(b, enc); err != nil {
		return err
	}

	op.BaseOperation = ubo

	var um MemoJSONUnpacker
	if err := enc.Unmarshal(b, &um); err != nil {
		return err
	}
	op.Memo = um.Memo

	return nil
}
