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

func NewAmountFromInterface(a interface{}) (Amount, error) {
	switch t := a.(type) {
	case int:
		return NewAmount(int64(t)), nil
	case int8:
		return NewAmount(int64(t)), nil
	case int32:
		return NewAmount(int64(t)), nil
	case int64:
		return NewAmount(t), nil
	case uint:
		return NewAmount(int64(t)), nil
	case uint8:
		return NewAmount(int64(t)), nil
	case uint32:
		return NewAmount(int64(t)), nil
	case uint64:
		return NewAmount(int64(t)), nil
	case string:
		if n, err := NewAmountFromString(t); err != nil {
			return NilAmount, xerrors.Errorf("invalid amount value, %q", t)
		} else {
			return n, nil
		}
	default:
		return NilAmount, xerrors.Errorf("unknown type of amount value, %T", a)
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

func (a Amount) MulInt64(b int64) Amount {
	i := big.NewInt(b)
	return NewAmountFromBigInt((new(big.Int)).Mul(a.Int, i))
}

func (a Amount) MulFloat64(b float64) Amount {
	af, _ := new(big.Float).SetString(a.Int.String())
	bf := big.NewFloat(b)

	c := new(big.Int)
	_, _ = new(big.Float).Mul(af, bf).Int(c)

	return NewAmountFromBigInt(c)
}

func (a Amount) Div(b Amount) Amount {
	return NewAmountFromBigInt((new(big.Int)).Div(a.Int, b.Int))
}

func (a Amount) Neg() Amount {
	return NewAmountFromBigInt((new(big.Int)).Neg(a.Int))
}
