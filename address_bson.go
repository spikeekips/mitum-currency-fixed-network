package mc

import (
	"go.mongodb.org/mongo-driver/bson"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

func (ca Address) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(bsonenc.NewHintedDoc(ca.Hint()),
			bson.M{
				"address": ca.String(),
			}))
}

type AddressBSONUnpacker struct {
	A string `bson:"address"`
}

func (ca *Address) UnmarshalBSON(b []byte) error {
	var uca AddressBSONUnpacker
	if err := bson.Unmarshal(b, &uca); err != nil {
		return err
	}

	return ca.unpack(nil, uca.A)
}
