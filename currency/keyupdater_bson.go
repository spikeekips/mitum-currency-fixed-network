package currency // nolint: dupl

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (fact KeyUpdaterFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(bsonenc.NewHintedDoc(fact.Hint()),
			bson.M{
				"hash":     fact.h,
				"token":    fact.token,
				"target":   fact.target,
				"keys":     fact.keys,
				"currency": fact.currency,
			}))
}

type KeyUpdaterFactBSONUnpacker struct {
	H  valuehash.Bytes     `bson:"hash"`
	TK []byte              `bson:"token"`
	TG base.AddressDecoder `bson:"target"`
	KS bson.Raw            `bson:"keys"`
	CR string              `bson:"currency"`
}

func (fact *KeyUpdaterFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ufact KeyUpdaterFactBSONUnpacker
	if err := bson.Unmarshal(b, &ufact); err != nil {
		return err
	}

	return fact.unpack(enc, ufact.H, ufact.TK, ufact.TG, ufact.KS, ufact.CR)
}

func (op *KeyUpdater) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ubo BaseOperation
	if err := ubo.UnpackBSON(b, enc); err != nil {
		return err
	}

	op.BaseOperation = ubo

	return nil
}
