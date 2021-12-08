package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	SuffrageInflationFactType   = hint.Type("mitum-currency-suffrage-inflation-operation-fact")
	SuffrageInflationFactHint   = hint.NewHint(SuffrageInflationFactType, "v0.0.1")
	SuffrageInflationFactHinter = SuffrageInflationFact{BaseHinter: hint.NewBaseHinter(SuffrageInflationFactHint)}
	SuffrageInflationType       = hint.Type("mitum-currency-suffrage-inflation-operation")
	SuffrageInflationHint       = hint.NewHint(SuffrageInflationType, "v0.0.1")
	SuffrageInflationHinter     = SuffrageInflation{BaseOperation: operationHinter(SuffrageInflationHint)}
)

type SuffrageInflationItem struct {
	receiver base.Address
	amount   Amount
}

func NewSuffrageInflationItem(receiver base.Address, amount Amount) SuffrageInflationItem {
	return SuffrageInflationItem{
		receiver: receiver,
		amount:   amount,
	}
}

func (item SuffrageInflationItem) Bytes() []byte {
	var br []byte
	if item.receiver != nil {
		br = item.receiver.Bytes()
	}

	return util.ConcatBytesSlice(br, item.amount.Bytes())
}

func (item SuffrageInflationItem) IsValid([]byte) error {
	if err := isvalid.Check(nil, false, item.receiver, item.amount); err != nil {
		return isvalid.InvalidError.Errorf("invalid SuffrageInflationItem: %w", err)
	}

	if !item.amount.Big().OverZero() {
		return isvalid.InvalidError.Errorf("under zero amount of SuffrageInflationItem")
	}

	return nil
}

type SuffrageInflationFact struct {
	hint.BaseHinter
	h     valuehash.Hash
	token []byte
	items []SuffrageInflationItem
}

func NewSuffrageInflationFact(token []byte, items []SuffrageInflationItem) SuffrageInflationFact {
	fact := SuffrageInflationFact{
		BaseHinter: hint.NewBaseHinter(SuffrageInflationFactHint),
		token:      token,
		items:      items,
	}

	fact.h = fact.GenerateHash()

	return fact
}

func (fact SuffrageInflationFact) Hash() valuehash.Hash {
	return fact.h
}

func (fact SuffrageInflationFact) Bytes() []byte {
	bi := make([][]byte, len(fact.items)+1)
	bi[0] = fact.token

	for i := range fact.items {
		bi[i+1] = fact.items[i].Bytes()
	}

	return util.ConcatBytesSlice(bi...)
}

func (fact SuffrageInflationFact) IsValid(b []byte) error {
	if err := IsValidOperationFact(fact, b); err != nil {
		return err
	}

	if len(fact.items) < 1 {
		return isvalid.InvalidError.Errorf("empty items for SuffrageInflationFact")
	}

	founds := map[string]struct{}{}
	for i := range fact.items {
		item := fact.items[i]
		if err := item.IsValid(nil); err != nil {
			return isvalid.InvalidError.Errorf("invalid SuffrageInflationFact: %w", err)
		}

		k := item.receiver.String() + "-" + item.amount.Currency().String()
		if _, found := founds[k]; found {
			return isvalid.InvalidError.Errorf("duplicated item found in SuffrageInflationFact")
		}
		founds[k] = struct{}{}
	}

	return nil
}

func (fact SuffrageInflationFact) GenerateHash() valuehash.Hash {
	return valuehash.NewSHA256(fact.Bytes())
}

func (fact SuffrageInflationFact) Token() []byte {
	return fact.token
}

func (fact SuffrageInflationFact) Items() []SuffrageInflationItem {
	return fact.items
}

type SuffrageInflation struct {
	BaseOperation
}

func NewSuffrageInflation(fact SuffrageInflationFact, fs []base.FactSign, memo string) (SuffrageInflation, error) {
	bo, err := NewBaseOperationFromFact(SuffrageInflationHint, fact, fs, memo)
	if err != nil {
		return SuffrageInflation{}, err
	}

	return SuffrageInflation{BaseOperation: bo}, nil
}
