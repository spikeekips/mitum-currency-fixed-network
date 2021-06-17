package currency

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/key"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type KeyJSONPacker struct {
	jsonenc.HintedHead
	W uint          `json:"weight"`
	K key.Publickey `json:"key"`
}

func (ky Key) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(KeyJSONPacker{
		HintedHead: jsonenc.NewHintedHead(ky.Hint()),
		W:          ky.w,
		K:          ky.k,
	})
}

type KeyJSONUnpacker struct {
	W uint                 `json:"weight"`
	K key.PublickeyDecoder `json:"key"`
}

func (ky *Key) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uk KeyJSONUnpacker
	if err := jsonenc.Unmarshal(b, &uk); err != nil {
		return err
	}

	return ky.unpack(enc, uk.W, uk.K)
}

type KeysJSONPacker struct {
	jsonenc.HintedHead
	H  valuehash.Hash `json:"hash"`
	KS []Key          `json:"keys"`
	TH uint           `json:"threshold"`
}

func (ks Keys) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(KeysJSONPacker{
		HintedHead: jsonenc.NewHintedHead(ks.Hint()),
		H:          ks.h,
		KS:         ks.keys,
		TH:         ks.threshold,
	})
}

type KeysJSONUnpacker struct {
	H  valuehash.Bytes `json:"hash"`
	KS json.RawMessage `json:"keys"`
	TH uint            `json:"threshold"`
}

func (ks *Keys) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uks KeysJSONUnpacker
	if err := jsonenc.Unmarshal(b, &uks); err != nil {
		return err
	}

	return ks.unpack(enc, uks.H, uks.KS, uks.TH)
}
