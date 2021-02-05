package currency

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson"
)

type AmountBSONPacker struct {
	CR CurrencyID `bson:"currency"`
	BG Big        `bson:"amount"`
}

func (am Amount) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(am.Hint()),
		bson.M{
			"currency": am.cid,
			"amount":   am.big,
		}),
	)
}

type AmountBSONUnpacker struct {
	CR string `bson:"currency"`
	BG Big    `bson:"amount"`
}

func (am *Amount) UnmarshalBSON(b []byte) error {
	var uam AmountBSONUnpacker
	if err := bsonenc.Unmarshal(b, &uam); err != nil {
		return err
	}

	am.big = uam.BG
	am.cid = CurrencyID(uam.CR)

	return nil
}
