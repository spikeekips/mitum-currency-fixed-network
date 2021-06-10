package currency

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

func (fa NilFeeer) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.NewHintedDoc(fa.Hint()))
}

func (*NilFeeer) UnmarsahlBSON() error {
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
	RC base.AddressDecoder `bson:"receiver"`
	AM Big                 `bson:"amount"`
}

func (fa *FixedFeeer) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ufa FixedFeeerBSONUnpacker
	if err := enc.Unmarshal(b, &ufa); err != nil {
		return err
	}

	return fa.unpack(enc, ufa.RC, ufa.AM)
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

	return fa.unpack(enc, ufa.RC, ufa.RA, ufa.MI, ufa.MA)
}
