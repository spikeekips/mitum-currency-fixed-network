package currency // nolint:dupl

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

func (it BaseTransfersItem) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(bsonenc.NewHintedDoc(it.Hint()),
			bson.M{
				"receiver": it.receiver,
				"amounts":  it.amounts,
			}),
	)
}

type BaseTransfersItemBSONUnpacker struct {
	RC base.AddressDecoder `bson:"receiver"`
	AM bson.Raw            `bson:"amounts"`
}

func (it *BaseTransfersItem) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ht bsonenc.HintedHead
	if err := enc.Unmarshal(b, &ht); err != nil {
		return err
	}

	var uit BaseTransfersItemBSONUnpacker
	if err := enc.Unmarshal(b, &uit); err != nil {
		return err
	}

	return it.unpack(enc, ht.H, uit.RC, uit.AM)
}
