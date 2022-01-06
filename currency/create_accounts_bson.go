package currency // nolint: dupl

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (fact CreateAccountsFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(bsonenc.NewHintedDoc(fact.Hint()),
			bson.M{
				"hash":   fact.h,
				"token":  fact.token,
				"sender": fact.sender,
				"items":  fact.items,
			}))
}

type CreateAccountsFactBSONUnpacker struct {
	H  valuehash.Bytes     `bson:"hash"`
	TK []byte              `bson:"token"`
	SD base.AddressDecoder `bson:"sender"`
	IT bson.Raw            `bson:"items"`
}

func (fact *CreateAccountsFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uca CreateAccountsFactBSONUnpacker
	if err := bson.Unmarshal(b, &uca); err != nil {
		return err
	}

	return fact.unpack(enc, uca.H, uca.TK, uca.SD, uca.IT)
}

func (op *CreateAccounts) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ubo BaseOperation
	if err := ubo.UnpackBSON(b, enc); err != nil {
		return err
	}

	op.BaseOperation = ubo

	return nil
}
