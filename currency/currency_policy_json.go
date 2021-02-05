package currency

import (
	"encoding/json"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type CurrencyPolicyJSONPacker struct {
	jsonenc.HintedHead
	MN Big   `json:"new_account_min_balance"`
	FE Feeer `json:"feeer"`
}

func (po CurrencyPolicy) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(CurrencyPolicyJSONPacker{
		HintedHead: jsonenc.NewHintedHead(po.Hint()),
		MN:         po.newAccountMinBalance,
		FE:         po.feeer,
	})
}

type CurrencyPolicyJSONUnpacker struct {
	MN Big             `json:"new_account_min_balance"`
	FE json.RawMessage `json:"feeer"`
}

func (po *CurrencyPolicy) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var upo CurrencyPolicyJSONUnpacker
	if err := enc.Unmarshal(b, &upo); err != nil {
		return err
	}

	return po.unpack(enc, upo.MN, upo.FE)
}
