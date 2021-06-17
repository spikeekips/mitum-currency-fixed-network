package currency // nolint:dupl

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson"
)

func (it BaseCreateAccountsItem) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(bsonenc.NewHintedDoc(it.Hint()),
			bson.M{
				"keys":    it.keys,
				"amounts": it.amounts,
			}),
	)
}

type CreateAccountsItemBSONUnpacker struct {
	KS bson.Raw `bson:"keys"`
	AM bson.Raw `bson:"amounts"`
}

func (it *BaseCreateAccountsItem) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ht bsonenc.HintedHead
	if err := enc.Unmarshal(b, &ht); err != nil {
		return err
	}

	var uca CreateAccountsItemBSONUnpacker
	if err := bson.Unmarshal(b, &uca); err != nil {
		return err
	}

	return it.unpack(enc, ht.H, uca.KS, uca.AM)
}
