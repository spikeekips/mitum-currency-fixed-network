package currency

import (
	"go.mongodb.org/mongo-driver/bson"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (fact FeeOperationFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(bsonenc.NewHintedDoc(fact.Hint()),
			bson.M{
				"hash":    fact.h,
				"token":   fact.token,
				"amounts": fact.amounts,
			}))
}

type FeeOperationFactBSONUnpacker struct {
	H  valuehash.Bytes `bson:"hash"`
	TK []byte          `bson:"token"`
	AM bson.Raw        `bson:"amounts"`
}

func (fact *FeeOperationFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uft FeeOperationFactBSONUnpacker
	if err := enc.Unmarshal(b, &uft); err != nil {
		return err
	}

	return fact.unpack(enc, uft.H, uft.TK, uft.AM)
}

func (op FeeOperation) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(op.Hint()),
		bson.M{
			"hash": op.h,
			"fact": op.fact,
		},
	))
}

type FeeOperationBSONUnpacker struct {
	H  valuehash.Bytes `bson:"hash"`
	FC bson.Raw        `bson:"fact"`
}

func (op *FeeOperation) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var upo FeeOperationBSONUnpacker
	if err := enc.Unmarshal(b, &upo); err != nil {
		return err
	}

	return op.unpack(enc, upo.H, upo.FC)
}
