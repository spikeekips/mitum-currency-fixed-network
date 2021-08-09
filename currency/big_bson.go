package currency

import (
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func (a Big) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, a.String()), nil
}

func (a *Big) UnmarshalBSONValue(t bsontype.Type, b []byte) error {
	if t != bsontype.String {
		return errors.Errorf("invalid marshaled type for Big, %v", t)
	}

	s, _, ok := bsoncore.ReadString(b)
	if !ok {
		return errors.Errorf("can not read string")
	}

	ua, err := NewBigFromString(s)
	if err != nil {
		return err
	}
	*a = ua

	return nil
}
