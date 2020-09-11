package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/state"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type AmountStatePackerJSON struct {
	state.StateV0PackerJSON
	jsonenc.HintedHead
	FE Amount `json:"fee"`
}

func (st AmountState) MarshalJSON() ([]byte, error) {
	var m state.StateV0PackerJSON
	if s, ok := st.State.(state.StateV0); !ok {
		return nil, xerrors.Errorf("unknown State, not state.StateV0, %T", st.State)
	} else {
		m = s.PackerJSON()
	}

	return jsonenc.Marshal(AmountStatePackerJSON{
		StateV0PackerJSON: m,
		HintedHead:        jsonenc.NewHintedHead(st.Hint()),
		FE:                st.fee,
	})
}

type AmountStateV0UnpackerJSON struct {
	FE Amount `json:"fee"`
}

func (st *AmountState) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	sst := new(state.StateV0)
	if err := sst.UnpackJSON(b, enc); err != nil {
		return err
	} else {
		st.State = *sst
	}

	var ust AmountStateV0UnpackerJSON
	if err := enc.Unmarshal(b, &ust); err != nil {
		return err
	} else {
		st.fee = ust.FE
	}

	return nil
}
