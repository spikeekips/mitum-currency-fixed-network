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

type TransferFact struct {
	h        valuehash.Hash
	token    []byte
	sender   base.Address
	receiver base.Address
	amount   Amount
}

func NewTransferFact(token []byte, sender, receiver base.Address, amount Amount) TransferFact {
	tff := TransferFact{
		token:    token,
		sender:   sender,
		receiver: receiver,
		amount:   amount,
	}
	tff.h = valuehash.NewSHA256(tff.Bytes())

	return tff
}

func (tff TransferFact) Hint() hint.Hint {
	return TransferFactHint
}

func (tff TransferFact) Hash() valuehash.Hash {
	return tff.h
}

func (tff TransferFact) Token() []byte {
	return tff.token
}

func (tff TransferFact) Bytes() []byte {
	return util.ConcatBytesSlice(
		tff.token,
		tff.sender.Bytes(),
		tff.receiver.Bytes(),
		tff.amount.Bytes(),
	)
}

func (tff TransferFact) IsValid([]byte) error {
	if len(tff.token) < 1 {
		return xerrors.Errorf("empty token for TransferFact")
	}

	if err := isvalid.Check([]isvalid.IsValider{
		tff.h,
		tff.sender,
		tff.receiver,
		tff.amount,
	}, nil, false); err != nil {
		return err
	}

	return nil
}

func (tff TransferFact) Sender() base.Address {
	return tff.sender
}

func (tff TransferFact) Receiver() base.Address {
	return tff.receiver
}

func (tff TransferFact) Amount() Amount {
	return tff.amount
}

type Transfer struct {
	operation.BaseOperation
	Memo string
}

func NewTransfer(
	fact TransferFact,
	fs []operation.FactSign,
	memo string,
) (Transfer, error) {
	if bo, err := operation.NewBaseOperationFromFact(TransferHint, fact, fs); err != nil {
		return Transfer{}, err
	} else {
		tf := Transfer{BaseOperation: bo, Memo: memo}

		tf.BaseOperation = bo.SetHash(tf.GenerateHash())

		return tf, nil
	}
}

func (tf Transfer) Hint() hint.Hint {
	return TransferHint
}

func (tf Transfer) IsValid(networkID []byte) error {
	if err := IsValidMemo(tf.Memo); err != nil {
		return err
	}

	return operation.IsValidOperation(tf, networkID)
}

func (tf Transfer) GenerateHash() valuehash.Hash {
	bs := make([][]byte, len(tf.Signs())+1)
	for i := range tf.Signs() {
		bs[i] = tf.Signs()[i].Bytes()
	}

	bs[len(bs)-1] = []byte(tf.Memo)

	e := util.ConcatBytesSlice(tf.Fact().Hash().Bytes(), util.ConcatBytesSlice(bs...))

	return valuehash.NewSHA256(e)
}

func (tf Transfer) AddFactSigns(fs ...operation.FactSign) (operation.FactSignUpdater, error) {
	if o, err := tf.BaseOperation.AddFactSigns(fs...); err != nil {
		return nil, err
	} else {
		tf.BaseOperation = o.(operation.BaseOperation)
	}

	tf.BaseOperation = tf.SetHash(tf.GenerateHash())

	return tf, nil
}

func (tf Transfer) Process(
	getState func(key string) (state.StateUpdater, bool, error),
	setState func(valuehash.Hash, ...state.StateUpdater) error,
) error {
	fact := tf.Fact().(TransferFact)

	if _, err := existsAccountState(StateKeyKeys(fact.sender), "keys of sender", getState); err != nil {
		return err
	}
	if _, err := existsAccountState(StateKeyKeys(fact.receiver), "keys of receiver", getState); err != nil {
		return err
	}

	var sBalance, rBalance state.StateUpdater
	{
		var err error
		if sBalance, err = existsAccountState(StateKeyBalance(fact.sender), "balance of sender", getState); err != nil {
			return err
		}
		if rBalance, err = existsAccountState(
			StateKeyBalance(fact.receiver), "balance of receiver", getState); err != nil {
			return err
		}
	}

	if err := checkFactSignsByState(fact.sender, tf.Signs(), getState); err != nil {
		return xerrors.Errorf("invalid signing: %w", err)
	}

	if b, err := StateAmountValue(sBalance); err != nil {
		return state.IgnoreOperationProcessingError.Wrap(err)
	} else {
		n := b.Sub(fact.amount)
		if err := n.IsValid(nil); err != nil {
			return state.IgnoreOperationProcessingError.Errorf("failed to sub amount from balance: %w", err)
		} else if err := SetStateAmountValue(sBalance, n); err != nil {
			return state.IgnoreOperationProcessingError.Wrap(err)
		}
	}

	if b, err := StateAmountValue(rBalance); err != nil {
		return state.IgnoreOperationProcessingError.Wrap(err)
	} else {
		n := b.Add(fact.amount)
		if err := n.IsValid(nil); err != nil {
			return state.IgnoreOperationProcessingError.Errorf("failed to add amount from balance: %w", err)
		} else if err := SetStateAmountValue(rBalance, n); err != nil {
			return state.IgnoreOperationProcessingError.Wrap(err)
		}
	}

	return setState(tf.Hash(), sBalance, rBalance)
}
