package currency

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (ac Account) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(ac.Hint()),
		bson.M{
			"hash":    ac.h,
			"address": ac.address,
			"keys":    ac.keys,
		},
	))
}

type AccountBSONUnpacker struct {
	H  valuehash.Bytes     `bson:"hash"`
	AD base.AddressDecoder `bson:"address"`
	KS bson.Raw            `bson:"keys"`
}

func (ac *Account) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uac AccountBSONUnpacker
	if err := enc.Unmarshal(b, &uac); err != nil {
		return err
	}

	return ac.unpack(enc, uac.H, uac.AD, uac.KS)
}
