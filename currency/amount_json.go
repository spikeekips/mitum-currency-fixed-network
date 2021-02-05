package currency

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type AmountJSONPacker struct {
	jsonenc.HintedHead
	BG Big        `json:"amount"`
	CR CurrencyID `json:"currency"`
}

func (am Amount) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(AmountJSONPacker{
		HintedHead: jsonenc.NewHintedHead(am.Hint()),
		BG:         am.big,
		CR:         am.cid,
	})
}

type AmountJSONUnpacker struct {
	BG Big    `json:"amount"`
	CR string `json:"currency"`
}

func (am *Amount) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uam AmountJSONUnpacker
	if err := enc.Unmarshal(b, &uam); err != nil {
		return err
	}

	am.big = uam.BG
	am.cid = CurrencyID(uam.CR)

	return nil
}
