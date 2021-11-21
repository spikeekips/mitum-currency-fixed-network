package currency

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type CurrencyDesignJSONPacker struct {
	jsonenc.HintedHead
	AM Amount         `json:"amount"`
	GA base.Address   `json:"genesis_account"`
	PO CurrencyPolicy `json:"policy"`
	AG Big            `json:"aggregate"`
}

func (de CurrencyDesign) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(CurrencyDesignJSONPacker{
		HintedHead: jsonenc.NewHintedHead(de.Hint()),
		AM:         de.Amount,
		GA:         de.genesisAccount,
		PO:         de.policy,
		AG:         de.aggregate,
	})
}

type CurrencyDesignJSONUnpacker struct {
	AM Amount              `json:"amount"`
	GA base.AddressDecoder `json:"genesis_account"`
	PO json.RawMessage     `json:"policy"`
	AG Big                 `json:"aggregate"`
}

func (de *CurrencyDesign) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ude CurrencyDesignJSONUnpacker
	if err := enc.Unmarshal(b, &ude); err != nil {
		return err
	}

	return de.unpack(enc, ude.AM, ude.GA, ude.PO, ude.AG)
}
