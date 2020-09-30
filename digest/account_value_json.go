package digest

import (
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type AccountValueJSONPacker struct {
	jsonenc.HintedHead
	currency.AccountPackerJSON
	BL currency.Amount `json:"balance"`
	HT base.Height     `json:"height"`
	PT base.Height     `json:"previous_height"`
}

func (va AccountValue) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(AccountValueJSONPacker{
		HintedHead:        jsonenc.NewHintedHead(va.Hint()),
		AccountPackerJSON: va.ac.PackerJSON(),
		BL:                va.balance,
		HT:                va.height,
		PT:                va.previousHeight,
	})
}

type AccountValueJSONUnpacker struct {
	BL currency.Amount `json:"balance"`
	HT base.Height     `json:"height"`
	PT base.Height     `json:"previous_height"`
}

func (va *AccountValue) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ua AccountValueJSONUnpacker
	if err := enc.Unmarshal(b, &ua); err != nil {
		return err
	} else {
		va.balance = ua.BL
		va.height = ua.HT
		va.previousHeight = ua.PT
	}

	var uac currency.Account
	if err := enc.Decode(b, &uac); err != nil {
		return err
	} else {
		va.ac = uac
	}

	return nil
}
