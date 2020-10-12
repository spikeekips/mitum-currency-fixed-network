package digest

import (
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/network"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

func (ni NodeInfo) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(bsonenc.NewHintedDoc(ni.Hint()), bson.M{
		"internal":        ni.NodeInfoV0,
		"fee_amount":      ni.feeAmount,
		"genesis_account": ni.genesisAccount,
		"genesis_balance": ni.genesisBalance,
	}))
}

type NodeInfoUnpackerBSON struct {
	FA string          `bson:"fee_amount"`
	GA bson.Raw        `bson:"genesis_account"`
	GB currency.Amount `bson:"genesis_balance"`
}

func (ni *NodeInfo) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	internal := new(network.NodeInfoV0)
	if err := internal.UnpackBSON(b, enc); err != nil {
		return err
	}

	var nni NodeInfoUnpackerBSON
	if err := enc.Unmarshal(b, &nni); err != nil {
		return err
	}

	var ga currency.Account
	if len(nni.GA) > 0 {
		if hinter, err := enc.DecodeByHint(nni.GA); err != nil {
			return err
		} else if i, ok := hinter.(currency.Account); !ok {
			return xerrors.Errorf("not Account in NodeInfo, %T", hinter)
		} else {
			ga = i
		}

		ni.genesisAccount = ga
		ni.genesisBalance = nni.GB
	}

	ni.NodeInfoV0 = *internal
	ni.feeAmount = nni.FA

	return nil
}
