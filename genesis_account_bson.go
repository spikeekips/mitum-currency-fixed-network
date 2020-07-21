package mc

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (gaf GenesisAccountFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(bsonenc.NewHintedDoc(gaf.Hint()),
			bson.M{
				"hash":             gaf.h,
				"token":            gaf.token,
				"genesis_node_key": gaf.genesisNodeKey,
				"keys":             gaf.keys,
				"amount":           gaf.amount,
			}))
}

type GenesisAccountFactBSONUnpacker struct {
	H  valuehash.Bytes      `bson:"hash"`
	TK []byte               `bson:"token"`
	GK key.PublickeyDecoder `bson:"genesis_node_key"`
	KS bson.Raw             `bson:"keys"`
	AM Amount               `bson:"amount"`
}

func (gaf *GenesisAccountFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uca GenesisAccountFactBSONUnpacker
	if err := bson.Unmarshal(b, &uca); err != nil {
		return err
	}

	return gaf.unpack(enc, uca.H, uca.TK, uca.GK, uca.KS, uca.AM)
}

func (ga GenesisAccount) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(ga.BaseOperation)
}

func (ga *GenesisAccount) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ubo operation.BaseOperation
	if err := ubo.UnpackBSON(b, enc); err != nil {
		return err
	}

	*ga = GenesisAccount{BaseOperation: ubo}

	return nil
}
