package currency

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (fact CurrencyPolicyUpdaterFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(bsonenc.NewHintedDoc(fact.Hint()),
			bson.M{
				"hash":     fact.h,
				"token":    fact.token,
				"currency": fact.cid,
				"policy":   fact.policy,
			}),
	)
}

type CurrencyPolicyUpdaterFactBSONUnpacker struct {
	H  valuehash.Bytes `bson:"hash"`
	TK []byte          `bson:"token"`
	CI string          `bson:"currency"`
	PO bson.Raw        `bson:"policy"`
}

func (fact *CurrencyPolicyUpdaterFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ufact CurrencyPolicyUpdaterFactBSONUnpacker
	if err := enc.Unmarshal(b, &ufact); err != nil {
		return err
	}

	return fact.unpack(enc, ufact.H, ufact.TK, ufact.CI, ufact.PO)
}

func (op *CurrencyPolicyUpdater) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ubo BaseOperation
	if err := ubo.UnpackBSON(b, enc); err != nil {
		return err
	}

	op.BaseOperation = ubo

	return nil
}
