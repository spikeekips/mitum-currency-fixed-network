package currency // nolint: dupl

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (cai CreateAccountItem) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bson.M{
		"keys":   cai.keys,
		"amount": cai.amount,
	})
}

type CreateAccountItemBSONUnpacker struct {
	KS bson.Raw `bson:"keys"`
	AM Amount   `bson:"amount"`
}

func (cai *CreateAccountItem) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uca CreateAccountItemBSONUnpacker
	if err := bson.Unmarshal(b, &uca); err != nil {
		return err
	}

	return cai.unpack(enc, uca.KS, uca.AM)
}

func (caf CreateAccountsFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(bsonenc.NewHintedDoc(caf.Hint()),
			bson.M{
				"hash":   caf.h,
				"token":  caf.token,
				"sender": caf.sender,
				"items":  caf.items,
			}))
}

type CreateAccountsFactBSONUnpacker struct {
	H  valuehash.Bytes     `bson:"hash"`
	TK []byte              `bson:"token"`
	SD base.AddressDecoder `bson:"sender"`
	IT []bson.Raw          `bson:"items"`
}

func (caf *CreateAccountsFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uca CreateAccountsFactBSONUnpacker
	if err := bson.Unmarshal(b, &uca); err != nil {
		return err
	}

	its := make([]CreateAccountItem, len(uca.IT))
	for i := range uca.IT {
		it := new(CreateAccountItem)
		if err := it.UnpackBSON(uca.IT[i], enc); err != nil {
			return err
		}

		its[i] = *it
	}

	return caf.unpack(enc, uca.H, uca.TK, uca.SD, its)
}

func (ca CreateAccounts) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(
			ca.BaseOperation.BSONM(),
			bson.M{"memo": ca.Memo},
		))
}

func (ca *CreateAccounts) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ubo operation.BaseOperation
	if err := ubo.UnpackBSON(b, enc); err != nil {
		return err
	}

	*ca = CreateAccounts{BaseOperation: ubo}

	var um MemoBSONUnpacker
	if err := enc.Unmarshal(b, &um); err != nil {
		return err
	} else {
		ca.Memo = um.Memo
	}

	return nil
}
