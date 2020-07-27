package currency // nolint: dupl

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (tff TransferFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(bsonenc.NewHintedDoc(tff.Hint()),
			bson.M{
				"hash":     tff.h,
				"token":    tff.token,
				"sender":   tff.sender,
				"receiver": tff.receiver,
				"amount":   tff.amount,
			}))
}

type TransferFactBSONUnpacker struct {
	H  valuehash.Bytes     `bson:"hash"`
	TK []byte              `bson:"token"`
	SD base.AddressDecoder `bson:"sender"`
	RC base.AddressDecoder `bson:"receiver"`
	AM Amount              `bson:"amount"`
}

func (tff *TransferFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var utff TransferFactBSONUnpacker
	if err := enc.Unmarshal(b, &utff); err != nil {
		return err
	}

	return tff.unpack(enc, utff.H, utff.TK, utff.SD, utff.RC, utff.AM)
}

func (tf Transfer) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(
			tf.BaseOperation.BSONM(),
			bson.M{"memo": tf.Memo},
		))
}

func (tf *Transfer) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ubo operation.BaseOperation
	if err := ubo.UnpackBSON(b, enc); err != nil {
		return err
	}

	*tf = Transfer{BaseOperation: ubo}

	var um MemoBSONUnpacker
	if err := enc.Unmarshal(b, &um); err != nil {
		return err
	} else {
		tf.Memo = um.Memo
	}

	return nil
}
