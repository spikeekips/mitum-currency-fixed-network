package currency

import (
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"
)

func (fact GenesisCurrenciesFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(bsonenc.NewHintedDoc(fact.Hint()),
			bson.M{
				"hash":             fact.h,
				"token":            fact.token,
				"genesis_node_key": fact.genesisNodeKey,
				"keys":             fact.keys,
				"currencies":       fact.cs,
			}))
}

type GenesisCurrenciesFactBSONUnpacker struct {
	H  valuehash.Bytes      `bson:"hash"`
	TK []byte               `bson:"token"`
	GK key.PublickeyDecoder `bson:"genesis_node_key"`
	KS bson.Raw             `bson:"keys"`
	CS []bson.Raw           `bson:"currencies"`
}

func (fact *GenesisCurrenciesFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ufact GenesisCurrenciesFactBSONUnpacker
	if err := bsonenc.Unmarshal(b, &ufact); err != nil {
		return xerrors.Errorf("failed to unmarshal GenesisCurrenciesFact: %w", err)
	}

	bcs := make([][]byte, len(ufact.CS))
	for i := range ufact.CS {
		bcs[i] = []byte(ufact.CS[i])
	}

	return fact.unpack(enc, ufact.H, ufact.TK, ufact.GK, ufact.KS, bcs)
}

func (op GenesisCurrencies) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(op.BaseOperation)
}

func (op *GenesisCurrencies) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ubo operation.BaseOperation
	if err := ubo.UnpackBSON(b, enc); err != nil {
		return err
	}

	*op = GenesisCurrencies{BaseOperation: ubo}

	return nil
}
