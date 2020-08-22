package currency // nolint: dupl

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (tff TransferItem) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bson.M{
		"receiver": tff.receiver,
		"amount":   tff.amount,
	})
}

type TransferItemBSONUnpacker struct {
	RC base.AddressDecoder `bson:"receiver"`
	AM Amount              `bson:"amount"`
}

func (tff *TransferItem) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var utff TransferItemBSONUnpacker
	if err := enc.Unmarshal(b, &utff); err != nil {
		return err
	}

	return tff.unpack(enc, utff.RC, utff.AM)
}

func (tff TransfersFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(bsonenc.NewHintedDoc(tff.Hint()),
			bson.M{
				"hash":   tff.h,
				"token":  tff.token,
				"sender": tff.sender,
				"items":  tff.items,
			}))
}

type TransferFactBSONUnpacker struct {
	H  valuehash.Bytes     `bson:"hash"`
	TK []byte              `bson:"token"`
	SD base.AddressDecoder `bson:"sender"`
	IT []bson.Raw          `bson:"items"`
}

func (tff *TransfersFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var utff TransferFactBSONUnpacker
	if err := enc.Unmarshal(b, &utff); err != nil {
		return err
	}

	its := make([]TransferItem, len(utff.IT))
	for i := range utff.IT {
		it := new(TransferItem)
		if err := it.UnpackBSON(utff.IT[i], enc); err != nil {
			return err
		}

		its[i] = *it
	}

	return tff.unpack(enc, utff.H, utff.TK, utff.SD, its)
}

func (tf Transfers) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(
			tf.BaseOperation.BSONM(),
			bson.M{"memo": tf.Memo},
		))
}

func (tf *Transfers) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ubo operation.BaseOperation
	if err := ubo.UnpackBSON(b, enc); err != nil {
		return err
	}

	*tf = Transfers{BaseOperation: ubo}

	var um MemoBSONUnpacker
	if err := enc.Unmarshal(b, &um); err != nil {
		return err
	} else {
		tf.Memo = um.Memo
	}

	return nil
}
