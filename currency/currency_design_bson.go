package currency

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

func (de CurrencyDesign) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(de.Hint()),
		bson.M{
			"amount":          de.Amount,
			"genesis_account": de.genesisAccount,
			"policy":          de.policy,
			"aggregate":       de.aggregate,
		}),
	)
}

type CurrencyDesignBSONUnpacker struct {
	AM Amount              `bson:"amount"`
	GA base.AddressDecoder `bson:"genesis_account"`
	PO bson.Raw            `bson:"policy"`
	AG Big                 `bson:"aggregate"`
}

func (de *CurrencyDesign) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ude CurrencyDesignBSONUnpacker
	if err := enc.Unmarshal(b, &ude); err != nil {
		return err
	}

	return de.unpack(enc, ude.AM, ude.GA, ude.PO, ude.AG)
}
