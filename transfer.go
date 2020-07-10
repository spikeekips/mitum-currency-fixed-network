package mc

import (
	"golang.org/x/xerrors"

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
	sender   Address
	receiver Address
	amount   Amount
}

func NewTransferFact(token []byte, sender, receiver Address, amount Amount) TransferFact {
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

type Transfer struct {
	operation.BaseOperation
}

func NewTransfer(
	fact TransferFact,
	fs []operation.FactSign,
) (Transfer, error) {
	if bo, err := operation.NewBaseOperationFromFact(TransferHint, fact, fs); err != nil {
		return Transfer{}, err
	} else {
		return Transfer{BaseOperation: bo}, nil
	}
}

func (tf Transfer) Hint() hint.Hint {
	return TransferHint
}

func (tf Transfer) IsValid(networkID []byte) error {
	return operation.IsValidOperation(tf, networkID)
}

func (tf Transfer) ProcessOperation(
	getState func(key string) (state.StateUpdater, bool, error),
	setState func(state.StateUpdater) error,
) error {
	fact := tf.Fact().(TransferFact)

	if _, err := loadState(StateKeyKeys(fact.sender), getState); err != nil {
		return xerrors.Errorf("keys of sender account does not exist: %w", err)
	}
	if _, err := loadState(StateKeyKeys(fact.receiver), getState); err != nil {
		return xerrors.Errorf("keys of receiver account does not exist: %w", err)
	}

	var sstateBalance, rstateBalance state.StateUpdater
	if st, err := loadState(StateKeyBalance(fact.sender), getState); err != nil {
		return xerrors.Errorf("balance of sender account does not exist: %w", err)
	} else {
		sstateBalance = st
	}
	if st, err := loadState(StateKeyBalance(fact.receiver), getState); err != nil {
		return xerrors.Errorf("balance of receiver account does not exist: %w", err)
	} else {
		rstateBalance = st
	}

	if err := checkFactSignsByState(fact.sender, tf.Signs(), getState); err != nil {
		return xerrors.Errorf("invalid signing: %w", err)
	}

	if b, err := StateAmountValue(sstateBalance); err != nil {
		return state.IgnoreOperationProcessingError.Wrap(err)
	} else {
		n := b.Sub(fact.amount)
		if err := n.IsValid(nil); err != nil {
			return state.IgnoreOperationProcessingError.Errorf("failed to sub amount from balance: %w", err)
		} else if err := SetStateAmountValue(sstateBalance, n); err != nil {
			return state.IgnoreOperationProcessingError.Wrap(err)
		}
	}

	if b, err := StateAmountValue(rstateBalance); err != nil {
		return state.IgnoreOperationProcessingError.Wrap(err)
	} else {
		n := b.Add(fact.amount)
		if err := n.IsValid(nil); err != nil {
			return state.IgnoreOperationProcessingError.Errorf("failed to add amount from balance: %w", err)
		} else if err := SetStateAmountValue(rstateBalance, n); err != nil {
			return state.IgnoreOperationProcessingError.Wrap(err)
		}
	}

	if err := setState(sstateBalance); err != nil {
		return err
	}
	if err := setState(rstateBalance); err != nil {
		return err
	}

	return nil
}
