package mc

import (
	"github.com/spikeekips/mitum/util/hint"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func (ca Address) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, hint.HintedString(ca.Hint(), ca.String())), nil
}
