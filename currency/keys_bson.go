package currency

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base/key"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (ky BaseAccountKey) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(ky.Hint()),
		bson.M{
			"weight": ky.w,
			"key":    ky.k,
		},
	))
}

type KeyBSONUnpacker struct {
	W uint                 `bson:"weight"`
	K key.PublickeyDecoder `bson:"key"`
}

func (ky *BaseAccountKey) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uk KeyBSONUnpacker
	if err := bson.Unmarshal(b, &uk); err != nil {
		return err
	}

	return ky.unpack(enc, uk.W, uk.K)
}

func (ks BaseAccountKeys) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(ks.Hint()),
		bson.M{
			"hash":      ks.h,
			"keys":      ks.keys,
			"threshold": ks.threshold,
		},
	))
}

type KeysBSONUnpacker struct {
	H  valuehash.Bytes `bson:"hash"`
	KS bson.Raw        `bson:"keys"`
	TH uint            `bson:"threshold"`
}

func (ks *BaseAccountKeys) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uks KeysBSONUnpacker
	if err := bson.Unmarshal(b, &uks); err != nil {
		return err
	}

	return ks.unpack(enc, uks.H, uks.KS, uks.TH)
}
