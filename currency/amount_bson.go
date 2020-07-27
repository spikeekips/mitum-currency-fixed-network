package currency

import (
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"golang.org/x/xerrors"
)

func (a Amount) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, a.String()), nil
}

func (a *Amount) UnmarshalBSONValue(t bsontype.Type, b []byte) error {
	if t != bsontype.String {
		return xerrors.Errorf("invalid marshaled type for Amount, %v", t)
	}

	s, _, ok := bsoncore.ReadString(b)
	if !ok {
		return xerrors.Errorf("can not read string")
	}

	if ua, err := NewAmountFromString(s); err != nil {
		return err
	} else {
		*a = ua
	}

	return nil
}
