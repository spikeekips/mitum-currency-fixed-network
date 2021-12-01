package currency

import (
	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func (fa NilFeeer) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		jsonenc.HintedHead
	}{
		HintedHead: jsonenc.NewHintedHead(fa.Hint()),
	})
}

func (*NilFeeer) UnmarsahlJSON() error {
	return nil
}

type FixedFeeerJSONPacker struct {
	jsonenc.HintedHead
	RC base.Address `json:"receiver"`
	AM Big          `json:"amount"`
}

func (fa FixedFeeer) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(FixedFeeerJSONPacker{
		HintedHead: jsonenc.NewHintedHead(fa.Hint()),
		RC:         fa.receiver,
		AM:         fa.amount,
	})
}

type FixedFeeerJSONUnpacker struct {
	RC base.AddressDecoder `json:"receiver"`
	AM Big                 `json:"amount"`
}

func (fa *FixedFeeer) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ufa FixedFeeerJSONUnpacker
	if err := enc.Unmarshal(b, &ufa); err != nil {
		return err
	}

	return fa.unpack(enc, ufa.RC, ufa.AM)
}

type RatioFeeerJSONPacker struct {
	jsonenc.HintedHead
	RC base.Address `json:"receiver"`
	RA float64      `json:"ratio"`
	MI Big          `json:"min"`
	MA Big          `json:"max"`
}

func (fa RatioFeeer) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(RatioFeeerJSONPacker{
		HintedHead: jsonenc.NewHintedHead(fa.Hint()),
		RC:         fa.receiver,
		RA:         fa.ratio,
		MI:         fa.min,
		MA:         fa.max,
	})
}

type RatioFeeerJSONUnpacker struct {
	RC base.AddressDecoder `json:"receiver"`
	RA float64             `json:"ratio"`
	MI Big                 `json:"min"`
	MA Big                 `json:"max"`
}

func (fa *RatioFeeer) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ufa RatioFeeerJSONUnpacker
	if err := enc.Unmarshal(b, &ufa); err != nil {
		return err
	}

	return fa.unpack(enc, ufa.RC, ufa.RA, ufa.MI, ufa.MA)
}
