package currency // nolint: dupl

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (caf CreateAccountFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(bsonenc.NewHintedDoc(caf.Hint()),
			bson.M{
				"hash":   caf.h,
				"token":  caf.token,
				"sender": caf.sender,
				"keys":   caf.keys,
				"amount": caf.amount,
			}))
}

type CreateAccountFactBSONUnpacker struct {
	H  valuehash.Bytes     `bson:"hash"`
	TK []byte              `bson:"token"`
	SD base.AddressDecoder `bson:"sender"`
	KS bson.Raw            `bson:"keys"`
	AM Amount              `bson:"amount"`
}

func (caf *CreateAccountFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uca CreateAccountFactBSONUnpacker
	if err := bson.Unmarshal(b, &uca); err != nil {
		return err
	}

	return caf.unpack(enc, uca.H, uca.TK, uca.SD, uca.KS, uca.AM)
}

func (ca CreateAccount) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(
			ca.BaseOperation.BSONM(),
			bson.M{"memo": ca.Memo},
		))
}

func (ca *CreateAccount) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ubo operation.BaseOperation
	if err := ubo.UnpackBSON(b, enc); err != nil {
		return err
	}

	*ca = CreateAccount{BaseOperation: ubo}

	var um MemoBSONUnpacker
	if err := enc.Unmarshal(b, &um); err != nil {
		return err
	} else {
		ca.Memo = um.Memo
	}

	return nil
}
