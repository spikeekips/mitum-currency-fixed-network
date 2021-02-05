package currency

import (
	"encoding/json"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type FeeOperationFactJSONPacker struct {
	jsonenc.HintedHead
	H  valuehash.Hash `json:"hash"`
	TK []byte         `json:"token"`
	AM []Amount       `json:"amounts"`
}

func (fact FeeOperationFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(FeeOperationFactJSONPacker{
		HintedHead: jsonenc.NewHintedHead(fact.Hint()),
		H:          fact.h,
		TK:         fact.token,
		AM:         fact.amounts,
	})
}

type FeeOperationFactJSONUnpacker struct {
	H  valuehash.Bytes   `json:"hash"`
	TK []byte            `json:"token"`
	AM []json.RawMessage `json:"amounts"`
}

func (fact *FeeOperationFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uft FeeOperationFactJSONUnpacker
	if err := enc.Unmarshal(b, &uft); err != nil {
		return err
	}

	bam := make([][]byte, len(uft.AM))
	for i := range uft.AM {
		bam[i] = uft.AM[i]
	}

	return fact.unpack(enc, uft.H, uft.TK, bam)
}

type FeeOperationJSONPacker struct {
	jsonenc.HintedHead
	H  valuehash.Hash   `json:"hash"`
	FT FeeOperationFact `json:"fact"`
}

func (op FeeOperation) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(FeeOperationJSONPacker{
		HintedHead: jsonenc.NewHintedHead(op.Hint()),
		H:          op.h,
		FT:         op.fact,
	})
}

type FeeOperationJSONUnpacker struct {
	H  valuehash.Bytes `json:"hash"`
	FT json.RawMessage `json:"fact"`
}

func (op *FeeOperation) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var upo FeeOperationJSONUnpacker
	if err := enc.Unmarshal(b, &upo); err != nil {
		return err
	}

	return op.unpack(enc, upo.H, upo.FT)
}
