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
		}),
	)
}

type CurrencyDesignBSONUnpacker struct {
	AM bson.Raw            `bson:"amount"`
	GA base.AddressDecoder `bson:"genesis_account"`
	PO bson.Raw            `bson:"policy"`
}

func (de *CurrencyDesign) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ude CurrencyDesignBSONUnpacker
	if err := enc.Unmarshal(b, &ude); err != nil {
		return err
	}

	return de.unpack(enc, ude.AM, ude.GA, ude.PO)
}
