package currency

import (
	"go.mongodb.org/mongo-driver/bson"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

func (po CurrencyPolicy) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(po.Hint()),
		bson.M{
			"new_account_min_balance": po.newAccountMinBalance,
			"feeer":                   po.feeer,
		}),
	)
}

type CurrencyPolicyBSONUnpacker struct {
	MN Big      `bson:"new_account_min_balance"`
	FE bson.Raw `bson:"feeer"`
}

func (po *CurrencyPolicy) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var upo CurrencyPolicyBSONUnpacker
	if err := enc.Unmarshal(b, &upo); err != nil {
		return err
	}

	return po.unpack(enc, upo.MN, upo.FE)
}
