package currency

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
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
	HT hint.Hint `json:"_hint"`
	BG Big       `json:"amount"`
	CR string    `json:"currency"`
}

func (am *Amount) UnmarshalJSON(b []byte) error {
	var uam AmountJSONUnpacker
	if err := jsonenc.Unmarshal(b, &uam); err != nil {
		return err
	}

	am.BaseHinter = hint.NewBaseHinter(uam.HT)
	am.big = uam.BG
	am.cid = CurrencyID(uam.CR)

	return nil
}
