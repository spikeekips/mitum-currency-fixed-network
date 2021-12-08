package currency

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func (ca Address) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, ca.String()), nil
}

func (ca *Address) UnpackBSON(b []byte, _ *bsonenc.Encoder) error {
	*ca = NewAddress(string(b))

	return nil
}
