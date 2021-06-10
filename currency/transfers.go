package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	TransfersFactType = hint.Type("mitum-currency-transfers-operation-fact")
	TransfersFactHint = hint.NewHint(TransfersFactType, "v0.0.1")
	TransfersType     = hint.Type("mitum-currency-transfers-operation")
	TransfersHint     = hint.NewHint(TransfersType, "v0.0.1")
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
	h      valuehash.Hash
	token  []byte
	sender base.Address
	items  []TransfersItem
}

func NewTransfersFact(token []byte, sender base.Address, items []TransfersItem) TransfersFact {
	fact := TransfersFact{
		token:  token,
		sender: sender,
		items:  items,
	}
	fact.h = fact.GenerateHash()

	return fact
}

func (TransfersFact) Hint() hint.Hint {
	return TransfersFactHint
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

func (fact TransfersFact) IsValid([]byte) error {
	if len(fact.token) < 1 {
		return xerrors.Errorf("empty token for TransferFact")
	} else if n := len(fact.items); n < 1 {
		return xerrors.Errorf("empty items")
	} else if n > int(MaxTransferItems) {
		return xerrors.Errorf("items, %d over max, %d", n, MaxTransferItems)
	}

	if err := isvalid.Check([]isvalid.IsValider{fact.h, fact.sender}, nil, false); err != nil {
		return err
	}

	foundReceivers := map[string]struct{}{}
	for i := range fact.items {
		it := fact.items[i]
		if err := it.IsValid(nil); err != nil {
			return xerrors.Errorf("invalid item found: %w", err)
		}

		k := StateAddressKeyPrefix(it.Receiver())
		switch _, found := foundReceivers[k]; {
		case found:
			return xerrors.Errorf("duplicated receiver found, %s", it.Receiver())
		case fact.sender.Equal(it.Receiver()):
			return xerrors.Errorf("receiver is same with sender, %q", fact.sender)
		default:
			foundReceivers[k] = struct{}{}
		}
	}

	if !fact.h.Equal(fact.GenerateHash()) {
		return isvalid.InvalidError.Errorf("wrong Fact hash")
	}

	return nil
}

func (fact TransfersFact) Sender() base.Address {
	return fact.sender
}

func (fact TransfersFact) Items() []TransfersItem {
	return fact.items
}

func (fact TransfersFact) Rebulild() TransfersFact {
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
	operation.BaseOperation
	Memo string
}

func NewTransfers(
	fact TransfersFact,
	fs []operation.FactSign,
	memo string,
) (Transfers, error) {
	bo, err := operation.NewBaseOperationFromFact(TransfersHint, fact, fs)
	if err != nil {
		return Transfers{}, err
	}
	op := Transfers{BaseOperation: bo, Memo: memo}

	op.BaseOperation = bo.SetHash(op.GenerateHash())

	return op, nil
}

func (Transfers) Hint() hint.Hint {
	return TransfersHint
}

func (op Transfers) IsValid(networkID []byte) error {
	if err := IsValidMemo(op.Memo); err != nil {
		return err
	}

	return operation.IsValidOperation(op, networkID)
}

func (op Transfers) GenerateHash() valuehash.Hash {
	bs := make([][]byte, len(op.Signs())+1)
	for i := range op.Signs() {
		bs[i] = op.Signs()[i].Bytes()
	}

	bs[len(bs)-1] = []byte(op.Memo)

	e := util.ConcatBytesSlice(op.Fact().Hash().Bytes(), util.ConcatBytesSlice(bs...))

	return valuehash.NewSHA256(e)
}

func (op Transfers) AddFactSigns(fs ...operation.FactSign) (operation.FactSignUpdater, error) {
	o, err := op.BaseOperation.AddFactSigns(fs...)
	if err != nil {
		return nil, err
	}
	op.BaseOperation = o.(operation.BaseOperation)

	op.BaseOperation = op.SetHash(op.GenerateHash())

	return op, nil
}
