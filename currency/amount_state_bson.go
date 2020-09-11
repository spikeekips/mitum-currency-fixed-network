package currency

import (
	"github.com/spikeekips/mitum/base/state"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"
)

func (st AmountState) MarshalBSON() ([]byte, error) {
	var m bson.M
	if s, ok := st.State.(state.StateV0); !ok {
		return nil, xerrors.Errorf("unknown State, not state.StateV0, %T", st.State)
	} else {
		m = s.BSONM()
	}

	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(st.Hint()),
		m,
		bson.M{
			"hash": st.Hash(),
			"fee":  st.fee,
		},
	))
}

type AmountStateUnpackerBSON struct {
	FE Amount `bson:"fee"`
}

func (st *AmountState) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	sst := new(state.StateV0)
	if err := sst.UnpackBSON(b, enc); err != nil {
		return err
	} else {
		st.State = *sst
	}

	var ust AmountStateUnpackerBSON
	if err := enc.Unmarshal(b, &ust); err != nil {
		return err
	}

	st.fee = ust.FE

	return nil
}
