package currency

import (
	"github.com/spikeekips/mitum/base/operation"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson"
)

func (op BaseOperation) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(
			op.BaseOperation.BSONM(),
			bson.M{"memo": op.Memo},
		))
}

func (op *BaseOperation) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ubo operation.BaseOperation
	if err := ubo.UnpackBSON(b, enc); err != nil {
		return err
	}

	op.BaseOperation = ubo

	var um MemoBSONUnpacker
	if err := enc.Unmarshal(b, &um); err != nil {
		return err
	}
	op.Memo = um.Memo

	return nil
}
