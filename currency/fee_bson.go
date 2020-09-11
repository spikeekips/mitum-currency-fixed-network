package currency

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (ft FeeOperationFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(bsonenc.NewHintedDoc(ft.Hint()),
			bson.M{
				"hash":       ft.h,
				"token":      ft.token,
				"fee_amount": ft.fa,
				"receiver":   ft.receiver,
				"fee":        ft.fee,
			}))
}

type FeeOperationFactBSONUnpacker struct {
	H  valuehash.Bytes     `bson:"hash"`
	TK []byte              `bson:"token"`
	FA string              `bson:"fee_amount"`
	RC base.AddressDecoder `bson:"receiver"`
	FE Amount              `bson:"fee"`
}

func (ft *FeeOperationFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uft FeeOperationFactBSONUnpacker
	if err := enc.Unmarshal(b, &uft); err != nil {
		return err
	}

	return ft.unpack(enc, uft.H, uft.TK, uft.FA, uft.RC, uft.FE)
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
