package currency

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type TransfersItemJSONPacker struct {
	jsonenc.HintedHead
	RC base.Address `json:"receiver"`
	AM []Amount     `json:"amounts"`
}

func (it BaseTransfersItem) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(TransfersItemJSONPacker{
		HintedHead: jsonenc.NewHintedHead(it.Hint()),
		RC:         it.receiver,
		AM:         it.amounts,
	})
}

type BaseTransfersItemJSONUnpacker struct {
	RC base.AddressDecoder `json:"receiver"`
	AM json.RawMessage     `json:"amounts"`
}

func (it *BaseTransfersItem) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uit BaseTransfersItemJSONUnpacker
	if err := enc.Unmarshal(b, &uit); err != nil {
		return err
	}

	return it.unpack(enc, uit.RC, uit.AM)
}
