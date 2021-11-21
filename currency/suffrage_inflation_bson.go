package currency

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (item SuffrageInflationItem) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(SuffrageInflationItemPacker{
		RC: item.receiver,
		AM: item.amount,
	})
}

func (item *SuffrageInflationItem) unpackBSON(b []byte, enc *bsonenc.Encoder) error {
	return item.unpack(b, enc)
}

func (fact SuffrageInflationFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(
		bsonenc.MergeBSONM(bsonenc.NewHintedDoc(fact.Hint()),
			bson.M{
				"hash":  fact.h,
				"token": fact.token,
				"items": fact.items,
			}),
	)
}

type SuffrageInflationFactBSONUnpacker struct {
	H  valuehash.Bytes `bson:"hash"`
	TK []byte          `bson:"token"`
	IS bson.Raw        `bson:"items"`
}

func (fact *SuffrageInflationFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uf SuffrageInflationFactBSONUnpacker
	if err := enc.Unmarshal(b, &uf); err != nil {
		return err
	}

	r, err := uf.IS.Values()
	if err != nil {
		return err
	}

	items := make([]SuffrageInflationItem, len(r))
	for i := range r {
		item := SuffrageInflationItem{}
		if err := item.unpackBSON(r[i].Value, enc); err != nil {
			return err
		}
		items[i] = item
	}

	return fact.unpack(enc, uf.H, uf.TK, items)
}

func (op *SuffrageInflation) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ubo BaseOperation
	if err := ubo.UnpackBSON(b, enc); err != nil {
		return err
	}

	op.BaseOperation = ubo

	return nil
}
