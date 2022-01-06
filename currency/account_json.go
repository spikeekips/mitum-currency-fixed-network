package currency

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type AccountPackerJSON struct {
	jsonenc.HintedHead
	H  valuehash.Hash `json:"hash"`
	AD base.Address   `json:"address"`
	KS AccountKeys    `json:"keys"`
}

func (ac Account) PackerJSON() AccountPackerJSON {
	return AccountPackerJSON{
		HintedHead: jsonenc.NewHintedHead(ac.Hint()),
		H:          ac.h,
		AD:         ac.address,
		KS:         ac.keys,
	}
}

func (ac Account) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(ac.PackerJSON())
}

type AccountJSONUnpacker struct {
	H  valuehash.Bytes     `json:"hash"`
	AD base.AddressDecoder `json:"address"`
	KS json.RawMessage     `json:"keys"`
}

func (ac *Account) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uac AccountJSONUnpacker
	if err := enc.Unmarshal(b, &uac); err != nil {
		return err
	}

	return ac.unpack(enc, uac.H, uac.AD, uac.KS)
}
