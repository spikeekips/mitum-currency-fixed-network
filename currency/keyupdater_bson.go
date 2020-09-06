package currency // nolint: dupl

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (ft KeyUpdaterFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(bsonenc.NewHintedDoc(ft.Hint()),
			bson.M{
				"hash":   ft.h,
				"token":  ft.token,
				"target": ft.target,
				"keys":   ft.keys,
			}))
}

type KeyUpdaterFactBSONUnpacker struct {
	H  valuehash.Bytes     `bson:"hash"`
	TK []byte              `bson:"token"`
	TG base.AddressDecoder `bson:"target"`
	KS bson.Raw            `bson:"keys"`
}

func (ft *KeyUpdaterFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var utf KeyUpdaterFactBSONUnpacker
	if err := bson.Unmarshal(b, &utf); err != nil {
		return err
	}

	return ft.unpack(enc, utf.H, utf.TK, utf.TG, utf.KS)
}

func (op KeyUpdater) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(
			op.BaseOperation.BSONM(),
			bson.M{"memo": op.Memo},
		))
}

func (op *KeyUpdater) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ubo operation.BaseOperation
	if err := ubo.UnpackBSON(b, enc); err != nil {
		return err
	}

	*op = KeyUpdater{BaseOperation: ubo}

	var um MemoBSONUnpacker
	if err := enc.Unmarshal(b, &um); err != nil {
		return err
	} else {
		op.Memo = um.Memo
	}

	return nil
}
