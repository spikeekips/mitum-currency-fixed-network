package currency

import (
	"encoding/json"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (item SuffrageInflationItem) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(SuffrageInflationItemPacker{
		RC: item.receiver,
		AM: item.amount,
	})
}

func (item *SuffrageInflationItem) unpackJSON(b []byte, enc *jsonenc.Encoder) error {
	return item.unpack(b, enc)
}

type SuffrageInflationFactJSONPacker struct {
	jsonenc.HintedHead
	H  valuehash.Hash          `json:"hash"`
	TK []byte                  `json:"token"`
	IS []SuffrageInflationItem `json:"items"`
}

func (fact SuffrageInflationFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(SuffrageInflationFactJSONPacker{
		HintedHead: jsonenc.NewHintedHead(fact.Hint()),
		H:          fact.h,
		TK:         fact.token,
		IS:         fact.items,
	})
}

type SuffrageInflationFactJSONUnpacker struct {
	H  valuehash.Bytes   `json:"hash"`
	TK []byte            `json:"token"`
	IS []json.RawMessage `json:"items"`
}

func (fact *SuffrageInflationFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uf SuffrageInflationFactJSONUnpacker
	if err := jsonenc.Unmarshal(b, &uf); err != nil {
		return err
	}

	items := make([]SuffrageInflationItem, len(uf.IS))
	for i := range uf.IS {
		item := SuffrageInflationItem{}
		if err := item.unpackJSON(uf.IS[i], enc); err != nil {
			return err
		}
		items[i] = item
	}

	return fact.unpack(enc, uf.H, uf.TK, items)
}

func (op *SuffrageInflation) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ubo BaseOperation
	if err := ubo.UnpackJSON(b, enc); err != nil {
		return err
	}

	op.BaseOperation = ubo

	return nil
}
