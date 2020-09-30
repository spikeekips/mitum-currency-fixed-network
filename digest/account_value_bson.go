package digest

import (
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"
)

func (va AccountValue) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(va.Hint()),
		bson.M{
			"ac":              va.ac,
			"balance":         va.balance,
			"height":          va.height,
			"previous_height": va.previousHeight,
		},
	))
}

type AccountValueBSONUnpacker struct {
	AC bson.Raw        `bson:"ac"`
	BL currency.Amount `bson:"balance"`
	HT base.Height     `bson:"height"`
	PT base.Height     `bson:"previous_height"`
}

func (va *AccountValue) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uva AccountValueBSONUnpacker
	if err := enc.Unmarshal(b, &uva); err != nil {
		return err
	}

	if hinter, err := enc.DecodeByHint(uva.AC); err != nil {
		return err
	} else if ac, ok := hinter.(currency.Account); !ok {
		return xerrors.Errorf("not currency.Account: %T", hinter)
	} else {
		va.ac = ac
	}

	va.balance = uva.BL
	va.height = uva.HT
	va.previousHeight = uva.PT

	return nil
}
