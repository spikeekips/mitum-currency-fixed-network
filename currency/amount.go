package currency

import (
	"math/big"

	"golang.org/x/xerrors"
)

var (
	NilAmount  = NewAmount(-1)
	ZeroAmount = NewAmount(0)
)

type Amount struct {
	*big.Int
}

func NewAmountFromBigInt(b *big.Int) Amount {
	return Amount{Int: b}
}

func NewAmount(i int64) Amount {
	return NewAmountFromBigInt(big.NewInt(i))
}

func NewAmountFromString(s string) (Amount, error) {
	if i, ok := new(big.Int).SetString(s, 10); !ok {
		return Amount{}, xerrors.Errorf("not proper Amount string, %q", s)
	} else {
		return NewAmountFromBigInt(i), nil
	}
}

func MustAmountFromString(s string) Amount {
	if i, ok := new(big.Int).SetString(s, 10); !ok {
		panic(xerrors.Errorf("not proper Amount string, %q", s))
	} else {
		return NewAmountFromBigInt(i)
	}
}

func (a Amount) IsZero() bool {
	return a.Int.Cmp(ZeroAmount.Int) == 0
}

func (a Amount) Equal(b Amount) bool {
	return a.Int.Cmp(b.Int) == 0
}

func (a Amount) Compare(b Amount) int {
	return a.Int.Cmp(b.Int)
}

func (a Amount) IsValid([]byte) error {
	if a.Compare(ZeroAmount) < 0 {
		return xerrors.Errorf("invalid amount; under zero")
	}

	return nil
}

func (a Amount) Add(b Amount) Amount {
	return NewAmountFromBigInt((new(big.Int)).Add(a.Int, b.Int))
}

func (a Amount) Sub(b Amount) Amount {
	return NewAmountFromBigInt((new(big.Int)).Sub(a.Int, b.Int))
}

func (a Amount) Mul(b Amount) Amount {
	return NewAmountFromBigInt((new(big.Int)).Mul(a.Int, b.Int))
}

func (a Amount) Div(b Amount) Amount {
	return NewAmountFromBigInt((new(big.Int)).Div(a.Int, b.Int))
}

func (a Amount) Neg() Amount {
	return NewAmountFromBigInt((new(big.Int)).Neg(a.Int))
}
