package digest

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base/state"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func LoadOperationHash(decoder func(interface{}) error) (valuehash.Hash, error) {
	var doc struct {
		FH valuehash.Bytes `bson:"fact"`
	}

	if err := decoder(&doc); err != nil {
		return nil, err
	}
	return doc.FH, nil
}

func LoadOperation(decoder func(interface{}) error, encs *encoder.Encoders) (OperationValue, error) {
	var b bson.Raw
	if err := decoder(&b); err != nil {
		return OperationValue{}, err
	}

	if _, hinter, err := mongodbstorage.LoadDataFromDoc(b, encs); err != nil {
		return OperationValue{}, err
	} else if va, ok := hinter.(OperationValue); !ok {
		return OperationValue{}, errors.Errorf("not OperationValue: %T", hinter)
	} else {
		return va, nil
	}
}

func LoadAccountValue(decoder func(interface{}) error, encs *encoder.Encoders) (AccountValue, error) {
	var b bson.Raw
	if err := decoder(&b); err != nil {
		return AccountValue{}, err
	}

	_, hinter, err := mongodbstorage.LoadDataFromDoc(b, encs)
	if err != nil {
		return AccountValue{}, err
	}

	rs, ok := hinter.(AccountValue)
	if !ok {
		return AccountValue{}, errors.Errorf("not AccountValue: %T", hinter)
	}

	return rs, nil
}

func LoadBalance(decoder func(interface{}) error, encs *encoder.Encoders) (state.State, error) {
	var b bson.Raw
	if err := decoder(&b); err != nil {
		return nil, err
	}

	if _, hinter, err := mongodbstorage.LoadDataFromDoc(b, encs); err != nil {
		return nil, err
	} else if st, ok := hinter.(state.State); !ok {
		return nil, errors.Errorf("not currency.Big: %T", hinter)
	} else {
		return st, nil
	}
}
