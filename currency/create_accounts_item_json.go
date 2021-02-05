package currency

import (
	"encoding/json"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type CreateAccountsItemJSONPacker struct {
	jsonenc.HintedHead
	KS Keys     `json:"keys"`
	AS []Amount `json:"amounts"`
}

func (it BaseCreateAccountsItem) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(CreateAccountsItemJSONPacker{
		HintedHead: jsonenc.NewHintedHead(it.Hint()),
		KS:         it.keys,
		AS:         it.amounts,
	})
}

type CreateAccountsItemJSONUnpacker struct {
	KS json.RawMessage   `json:"keys"`
	AM []json.RawMessage `json:"amounts"`
}

func (it *BaseCreateAccountsItem) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ht jsonenc.HintedHead
	if err := enc.Unmarshal(b, &ht); err != nil {
		return err
	}

	var uca CreateAccountsItemJSONUnpacker
	if err := jsonenc.Unmarshal(b, &uca); err != nil {
		return err
	}

	bam := make([][]byte, len(uca.AM))
	for i := range uca.AM {
		bam[i] = uca.AM[i]
	}

	return it.unpack(enc, ht.H, uca.KS, bam)
}
