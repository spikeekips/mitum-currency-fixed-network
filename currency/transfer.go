package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	TransferFactType = hint.MustNewType(0xa0, 0x01, "mitum-currency-transfer-operation-fact")
	TransferFactHint = hint.MustHint(TransferFactType, "0.0.1")
	TransferType     = hint.MustNewType(0xa0, 0x02, "mitum-currency-transfer-operation")
	TransferHint     = hint.MustHint(TransferType, "0.0.1")
)

var maxTransferItems uint = 10

type TransferItem struct {
	receiver base.Address
	amount   Amount
}

func NewTransferItem(receiver base.Address, amount Amount) TransferItem {
	return TransferItem{
		receiver: receiver,
		amount:   amount,
	}
}

func (tff TransferItem) Bytes() []byte {
	return util.ConcatBytesSlice(
		tff.receiver.Bytes(),
		tff.amount.Bytes(),
	)
}

func (tff TransferItem) IsValid([]byte) error {
	if err := isvalid.Check([]isvalid.IsValider{tff.receiver, tff.amount}, nil, false); err != nil {
		return err
	}

	if tff.amount.IsZero() {
		return xerrors.Errorf("amount should be over zero")
	}

	return nil
}

func (tff TransferItem) Receiver() base.Address {
	return tff.receiver
}

func (tff TransferItem) Amount() Amount {
	return tff.amount
}

type TransfersFact struct {
	h      valuehash.Hash
	token  []byte
	sender base.Address
	items  []TransferItem
}

func NewTransfersFact(token []byte, sender base.Address, items []TransferItem) TransfersFact {
	tff := TransfersFact{
		token:  token,
		sender: sender,
		items:  items,
	}
	tff.h = valuehash.NewSHA256(tff.Bytes())

	return tff
}

func (tff TransfersFact) Hint() hint.Hint {
	return TransferFactHint
}

func (tff TransfersFact) Hash() valuehash.Hash {
	return tff.h
}

func (tff TransfersFact) Token() []byte {
	return tff.token
}

func (tff TransfersFact) Bytes() []byte {
	its := make([][]byte, len(tff.items))
	for i := range tff.items {
		its[i] = tff.items[i].Bytes()
	}

	return util.ConcatBytesSlice(
		tff.token,
		tff.sender.Bytes(),
		util.ConcatBytesSlice(its...),
	)
}

func (tff TransfersFact) IsValid([]byte) error {
	if len(tff.token) < 1 {
		return xerrors.Errorf("empty token for TransferFact")
	} else if n := len(tff.items); n < 1 {
		return xerrors.Errorf("empty items")
	} else if n > int(maxTransferItems) {
		return xerrors.Errorf("items, %d over max, %d", n, maxTransferItems)
	}

	if err := isvalid.Check([]isvalid.IsValider{tff.h, tff.sender}, nil, false); err != nil {
		return err
	}

	foundReceivers := map[string]struct{}{}
	for i := range tff.items {
		it := tff.items[i]
		if err := it.IsValid(nil); err != nil {
			return err
		}

		k := it.Receiver().String()
		switch _, found := foundReceivers[k]; {
		case found:
			return xerrors.Errorf("duplicated receiver found, %s", k)
		case tff.sender.Equal(it.Receiver()):
			return xerrors.Errorf("receiver is same with sender, %q", tff.sender)
		default:
			foundReceivers[k] = struct{}{}
		}
	}

	return nil
}

func (tff TransfersFact) Sender() base.Address {
	return tff.sender
}

func (tff TransfersFact) Items() []TransferItem {
	return tff.items
}

func (tff TransfersFact) Amount() Amount {
	a := NewAmount(0)
	for i := range tff.items {
		a = a.Add(tff.items[i].Amount())
	}

	return a
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
	if bo, err := operation.NewBaseOperationFromFact(TransferHint, fact, fs); err != nil {
		return Transfers{}, err
	} else {
		tf := Transfers{BaseOperation: bo, Memo: memo}

		tf.BaseOperation = bo.SetHash(tf.GenerateHash())

		return tf, nil
	}
}

func (tf Transfers) Hint() hint.Hint {
	return TransferHint
}

func (tf Transfers) IsValid(networkID []byte) error {
	if err := IsValidMemo(tf.Memo); err != nil {
		return err
	}

	return operation.IsValidOperation(tf, networkID)
}

func (tf Transfers) GenerateHash() valuehash.Hash {
	bs := make([][]byte, len(tf.Signs())+1)
	for i := range tf.Signs() {
		bs[i] = tf.Signs()[i].Bytes()
	}

	bs[len(bs)-1] = []byte(tf.Memo)

	e := util.ConcatBytesSlice(tf.Fact().Hash().Bytes(), util.ConcatBytesSlice(bs...))

	return valuehash.NewSHA256(e)
}

func (tf Transfers) AddFactSigns(fs ...operation.FactSign) (operation.FactSignUpdater, error) {
	if o, err := tf.BaseOperation.AddFactSigns(fs...); err != nil {
		return nil, err
	} else {
		tf.BaseOperation = o.(operation.BaseOperation)
	}

	tf.BaseOperation = tf.SetHash(tf.GenerateHash())

	return tf, nil
}

func (tf Transfers) Process(
	func(key string) (state.State, bool, error),
	func(valuehash.Hash, ...state.State) error,
) error {
	// NOTE Process is nil func
	return nil
}

type TransferProcessor struct {
	h valuehash.Hash

	fact TransferItem

	rb AmountState
}

func (tf *TransferProcessor) PreProcess(
	getState func(key string) (state.State, bool, error),
	_ func(valuehash.Hash, ...state.State) error,
) error {
	if _, err := existsAccountState(StateKeyKeys(tf.fact.receiver), "keys of receiver", getState); err != nil {
		return err
	}

	if st, err := existsAccountState(StateKeyBalance(tf.fact.receiver), "balance of receiver", getState); err != nil {
		return err
	} else if ast, ok := st.(AmountState); !ok {
		return xerrors.Errorf("expected AmountState, but %T", st)
	} else {
		tf.rb = ast
	}

	return nil
}

func (tf *TransferProcessor) Process(
	_ func(key string) (state.State, bool, error),
	_ func(valuehash.Hash, ...state.State) error,
) (state.State, error) {
	return tf.rb.Add(tf.fact.Amount()), nil
}

type TransfersProcessor struct {
	Transfers

	sb AmountState
	rb []*TransferProcessor
}

func (tf *TransfersProcessor) PreProcess(
	getState func(key string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) (state.Processor, error) {
	fact := tf.Fact().(TransfersFact)

	if err := checkExistsAccountState(StateKeyKeys(fact.sender), getState); err != nil {
		return nil, err
	}

	if st, err := existsAccountState(StateKeyBalance(fact.sender), "balance of sender", getState); err != nil {
		return nil, err
	} else if b, err := StateAmountValue(st); err != nil {
		return nil, state.IgnoreOperationProcessingError.Wrap(err)
	} else if b.Compare(fact.Amount()) < 0 {
		return nil, state.IgnoreOperationProcessingError.Errorf("insufficient balance of sender")
	} else if ast, ok := st.(AmountState); !ok {
		return nil, xerrors.Errorf("expected AmountState, but %T", st)
	} else {
		tf.sb = ast
	}

	rb := make([]*TransferProcessor, len(fact.items))
	for i := range fact.items {
		c := &TransferProcessor{h: tf.Hash(), fact: fact.items[i]}
		if err := c.PreProcess(getState, setState); err != nil {
			return nil, state.IgnoreOperationProcessingError.Wrap(err)
		}

		rb[i] = c
	}

	if err := checkFactSignsByState(fact.sender, tf.Signs(), getState); err != nil {
		return nil, xerrors.Errorf("invalid signing: %w", err)
	}

	tf.rb = rb

	return tf, nil
}

func (tf *TransfersProcessor) Process(
	getState func(key string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) error {
	fact := tf.Fact().(TransfersFact)

	sts := make([]state.State, len(tf.rb)+1)
	for i := range tf.rb {
		if st, err := tf.rb[i].Process(getState, setState); err != nil {
			return state.IgnoreOperationProcessingError.Errorf("failed to process transfer item: %w", err)
		} else {
			sts[i] = st
		}
	}

	sts[len(sts)-1] = tf.sb.Sub(fact.Amount())

	return setState(tf.Hash(), sts...)
}
