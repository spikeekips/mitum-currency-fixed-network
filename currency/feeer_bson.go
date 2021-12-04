package currency

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/hint"
)

func (fa NilFeeer) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.NewHintedDoc(fa.Hint()))
}

func (fa *NilFeeer) UnmarsahlBSON(b []byte) error {
	var ht bsonenc.HintedHead
	if err := bsonenc.Unmarshal(b, &ht); err != nil {
		return err
	}

	fa.BaseHinter = hint.NewBaseHinter(ht.H)

	return nil
}

func (fa FixedFeeer) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(fa.Hint()),
		bson.M{
			"receiver": fa.receiver,
			"amount":   fa.amount,
		}),
	)
}

type FixedFeeerBSONUnpacker struct {
	HT hint.Hint           `bson:"_hint"`
	RC base.AddressDecoder `bson:"receiver"`
	AM Big                 `bson:"amount"`
}

func (fa *FixedFeeer) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ufa FixedFeeerBSONUnpacker
	if err := enc.Unmarshal(b, &ufa); err != nil {
		return err
	}

	return fa.unpack(enc, ufa.HT, ufa.RC, ufa.AM)
}

func (fa RatioFeeer) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(fa.Hint()),
		bson.M{
			"receiver": fa.receiver,
			"ratio":    fa.ratio,
			"min":      fa.min,
			"max":      fa.max,
		}),
	)
}

type RatioFeeerBSONUnpacker struct {
	HT hint.Hint           `bson:"_hint"`
	RC base.AddressDecoder `bson:"receiver"`
	RA float64             `bson:"ratio"`
	MI Big                 `bson:"min"`
	MA Big                 `bson:"max"`
}

func (fa *RatioFeeer) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ufa RatioFeeerBSONUnpacker
	if err := enc.Unmarshal(b, &ufa); err != nil {
		return err
	}

	return fa.unpack(enc, ufa.HT, ufa.RC, ufa.RA, ufa.MI, ufa.MA)
}
