package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	TransfersFactType   = hint.Type("mitum-currency-transfers-operation-fact")
	TransfersFactHint   = hint.NewHint(TransfersFactType, "v0.0.1")
	TransfersFactHinter = TransfersFact{BaseHinter: hint.NewBaseHinter(TransfersFactHint)}
	TransfersType       = hint.Type("mitum-currency-transfers-operation")
	TransfersHint       = hint.NewHint(TransfersType, "v0.0.1")
	TransfersHinter     = Transfers{BaseOperation: operationHinter(TransfersHint)}
)

var MaxTransferItems uint = 10

type TransfersItem interface {
	hint.Hinter
	isvalid.IsValider
	AmountsItem
	Bytes() []byte
	Receiver() base.Address
	Rebuild() TransfersItem
}

type TransfersFact struct {
	hint.BaseHinter
	h      valuehash.Hash
	token  []byte
	sender base.Address
	items  []TransfersItem
}

func NewTransfersFact(token []byte, sender base.Address, items []TransfersItem) TransfersFact {
	fact := TransfersFact{
		BaseHinter: hint.NewBaseHinter(TransfersFactHint),
		token:      token,
		sender:     sender,
		items:      items,
	}
	fact.h = fact.GenerateHash()

	return fact
}

func (fact TransfersFact) Hash() valuehash.Hash {
	return fact.h
}

func (fact TransfersFact) GenerateHash() valuehash.Hash {
	return valuehash.NewSHA256(fact.Bytes())
}

func (fact TransfersFact) Token() []byte {
	return fact.token
}

func (fact TransfersFact) Bytes() []byte {
	its := make([][]byte, len(fact.items))
	for i := range fact.items {
		its[i] = fact.items[i].Bytes()
	}

	return util.ConcatBytesSlice(
		fact.token,
		fact.sender.Bytes(),
		util.ConcatBytesSlice(its...),
	)
}

func (fact TransfersFact) IsValid(b []byte) error {
	if err := IsValidOperationFact(fact, b); err != nil {
		return err
	}

	if n := len(fact.items); n < 1 {
		return isvalid.InvalidError.Errorf("empty items")
	} else if n > int(MaxTransferItems) {
		return isvalid.InvalidError.Errorf("items, %d over max, %d", n, MaxTransferItems)
	}

	if err := isvalid.Check(nil, false, fact.sender); err != nil {
		return err
	}

	foundReceivers := map[string]struct{}{}
	for i := range fact.items {
		it := fact.items[i]
		if err := isvalid.Check(nil, false, it); err != nil {
			return isvalid.InvalidError.Errorf("invalid item found: %w", err)
		}

		k := it.Receiver().String()
		switch _, found := foundReceivers[k]; {
		case found:
			return isvalid.InvalidError.Errorf("duplicated receiver found, %s", it.Receiver())
		case fact.sender.Equal(it.Receiver()):
			return isvalid.InvalidError.Errorf("receiver is same with sender, %q", fact.sender)
		default:
			foundReceivers[k] = struct{}{}
		}
	}

	return nil
}

func (fact TransfersFact) Sender() base.Address {
	return fact.sender
}

func (fact TransfersFact) Items() []TransfersItem {
	return fact.items
}

func (fact TransfersFact) Rebuild() TransfersFact {
	items := make([]TransfersItem, len(fact.items))
	for i := range fact.items {
		it := fact.items[i]
		items[i] = it.Rebuild()
	}

	fact.items = items
	fact.h = fact.GenerateHash()

	return fact
}

func (fact TransfersFact) Addresses() ([]base.Address, error) {
	as := make([]base.Address, len(fact.items)+1)
	for i := range fact.items {
		as[i] = fact.items[i].Receiver()
	}

	as[len(fact.items)] = fact.Sender()

	return as, nil
}

type Transfers struct {
	BaseOperation
}

func NewTransfers(
	fact TransfersFact,
	fs []base.FactSign,
	memo string,
) (Transfers, error) {
	bo, err := NewBaseOperationFromFact(TransfersHint, fact, fs, memo)
	if err != nil {
		return Transfers{}, err
	}
	return Transfers{BaseOperation: bo}, nil
}
