package currency

import (
	"fmt"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	AmountType = hint.Type("mitum-currency-amount")
	AmountHint = hint.NewHint(AmountType, "v0.0.1")
)

type Amount struct {
	big Big
	cid CurrencyID
}

func NewAmount(big Big, cid CurrencyID) Amount {
	am := Amount{big: big, cid: cid}

	return am
}

func NewZeroAmount(cid CurrencyID) Amount {
	return NewAmount(NewBig(0), cid)
}

func MustNewAmount(big Big, cid CurrencyID) Amount {
	am := NewAmount(big, cid)
	if err := am.IsValid(nil); err != nil {
		panic(err)
	}

	return am
}

func (Amount) Hint() hint.Hint {
	return AmountHint
}

func (am Amount) Bytes() []byte {
	return util.ConcatBytesSlice(
		am.big.Bytes(),
		am.cid.Bytes(),
	)
}

func (am Amount) Hash() valuehash.Hash {
	return am.GenerateHash()
}

func (am Amount) GenerateHash() valuehash.Hash {
	return valuehash.NewSHA256(am.Bytes())
}

func (am Amount) IsEmpty() bool {
	return len(am.cid) < 1 || !am.big.OverNil()
}

func (am Amount) IsValid([]byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		am.cid,
		am.big,
	}, nil, false); err != nil {
		return xerrors.Errorf("invalid Balance: %w", err)
	}

	return nil
}

func (am Amount) Big() Big {
	return am.big
}

func (am Amount) Currency() CurrencyID {
	return am.cid
}

func (am Amount) String() string {
	return fmt.Sprintf("%s(%s)", am.big.String(), am.cid)
}

func (am Amount) Equal(b Amount) bool {
	switch {
	case am.cid != b.cid:
		return false
	case !am.big.Equal(b.big):
		return false
	default:
		return true
	}
}

func (am Amount) WithBig(big Big) Amount {
	am.big = big

	return am
}
