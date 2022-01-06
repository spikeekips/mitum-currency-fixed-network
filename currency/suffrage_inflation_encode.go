package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

type SuffrageInflationItemPacker struct {
	RC base.Address `bson:"receiver" json:"receiver"`
	AM Amount       `bson:"amount" json:"amount"`
}

type SuffrageInflationItemUnpacker struct {
	RC base.AddressDecoder `bson:"receiver" json:"receiver"`
	AM Amount              `bson:"amount" json:"amount"`
}

func (item *SuffrageInflationItem) unpack(b []byte, enc encoder.Encoder) error {
	var ui SuffrageInflationItemUnpacker
	if err := enc.Unmarshal(b, &ui); err != nil {
		return err
	}

	receiver, err := ui.RC.Encode(enc)
	if err != nil {
		return err
	}

	item.receiver = receiver
	item.amount = ui.AM

	return nil
}

func (fact *SuffrageInflationFact) unpack(
	_ encoder.Encoder,
	h valuehash.Hash,
	token []byte,
	items []SuffrageInflationItem,
) error {
	fact.h = h
	fact.token = token
	fact.items = items

	return nil
}
