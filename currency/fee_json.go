package currency

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type FeeOperationFactJSONPacker struct {
	jsonenc.HintedHead
	H  valuehash.Hash `json:"hash"`
	TK []byte         `json:"token"`
	FA string         `json:"fee_amount"`
	RC base.Address   `json:"receiver"`
	FE Amount         `json:"fee"`
}

func (ft FeeOperationFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(FeeOperationFactJSONPacker{
		HintedHead: jsonenc.NewHintedHead(ft.Hint()),
		H:          ft.h,
		TK:         ft.token,
		FA:         ft.fa,
		RC:         ft.receiver,
		FE:         ft.fee,
	})
}

type FeeOperationFactJSONUnpacker struct {
	H  valuehash.Bytes     `json:"hash"`
	TK []byte              `json:"token"`
	FA string              `json:"fee_amount"`
	RC base.AddressDecoder `json:"receiver"`
	FE Amount              `json:"fee"`
}

func (ft *FeeOperationFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uft FeeOperationFactJSONUnpacker
	if err := enc.Unmarshal(b, &uft); err != nil {
		return err
	}

	return ft.unpack(enc, uft.H, uft.TK, uft.FA, uft.RC, uft.FE)
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
	FC json.RawMessage `json:"fact"`
}

func (op *FeeOperation) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var upo FeeOperationJSONUnpacker
	if err := enc.Unmarshal(b, &upo); err != nil {
		return err
	}

	return op.unpack(enc, upo.H, upo.FC)
}
