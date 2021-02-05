package currency

import jsonenc "github.com/spikeekips/mitum/util/encoder/json"

func (st AmountState) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(st.State)
}
