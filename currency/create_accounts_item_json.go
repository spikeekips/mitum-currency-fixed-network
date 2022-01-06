package currency

import (
	"encoding/json"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type CreateAccountsItemJSONPacker struct {
	jsonenc.HintedHead
	KS AccountKeys `json:"keys"`
	AS []Amount    `json:"amounts"`
}

func (it BaseCreateAccountsItem) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(CreateAccountsItemJSONPacker{
		HintedHead: jsonenc.NewHintedHead(it.Hint()),
		KS:         it.keys,
		AS:         it.amounts,
	})
}

type CreateAccountsItemJSONUnpacker struct {
	KS json.RawMessage `json:"keys"`
	AM json.RawMessage `json:"amounts"`
}

func (it *BaseCreateAccountsItem) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uca CreateAccountsItemJSONUnpacker
	if err := jsonenc.Unmarshal(b, &uca); err != nil {
		return err
	}

	return it.unpack(enc, uca.KS, uca.AM)
}
