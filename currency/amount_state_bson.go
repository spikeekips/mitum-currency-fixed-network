package currency

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

func (st AmountState) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(st.State)
}
