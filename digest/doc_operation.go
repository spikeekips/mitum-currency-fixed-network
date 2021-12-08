package digest

import (
	"time"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

type OperationDoc struct {
	mongodbstorage.BaseDoc
	va        OperationValue
	op        operation.Operation
	addresses []string
	height    base.Height
}

func NewOperationDoc(
	op operation.Operation,
	enc encoder.Encoder,
	height base.Height,
	confirmedAt time.Time,
	inState bool,
	reason operation.ReasonError,
	index uint64,
) (OperationDoc, error) {
	var addresses []string
	if ads, ok := op.Fact().(currency.Addresses); ok {
		as, err := ads.Addresses()
		if err != nil {
			return OperationDoc{}, err
		}
		addresses = make([]string, len(as))
		for i := range as {
			addresses[i] = as[i].String()
		}
	}

	va := NewOperationValue(op, height, confirmedAt, inState, reason, index)
	b, err := mongodbstorage.NewBaseDoc(nil, va, enc)
	if err != nil {
		return OperationDoc{}, err
	}

	return OperationDoc{
		BaseDoc:   b,
		va:        va,
		op:        op,
		addresses: addresses,
		height:    height,
	}, nil
}

func (doc OperationDoc) MarshalBSON() ([]byte, error) {
	m, err := doc.BaseDoc.M()
	if err != nil {
		return nil, err
	}

	m["addresses"] = doc.addresses
	m["fact"] = doc.op.Fact().Hash()
	m["height"] = doc.height
	m["index"] = doc.va.index

	return bsonenc.Marshal(m)
}
