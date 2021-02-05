package digest

import (
	"github.com/spikeekips/mitum/network"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson"
)

func (ni NodeInfo) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(bsonenc.NewHintedDoc(ni.Hint()), bson.M{
		"internal": ni.NodeInfoV0,
	}))
}

func (ni *NodeInfo) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	internal := new(network.NodeInfoV0)
	if err := internal.UnpackBSON(b, enc); err != nil {
		return err
	}

	ni.NodeInfoV0 = *internal

	return nil
}
