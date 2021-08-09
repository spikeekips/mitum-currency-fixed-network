package currency

import (
	"math/big"

	"github.com/pkg/errors"
)

var (
	NilBig       = NewBig(-1)
	NilBigString = big.NewInt(-1).String()
	ZeroBig      = NewBig(0)
)

type Big struct {
	*big.Int
}

func NewBigFromBigInt(b *big.Int) Big {
	return Big{Int: b}
}

func NewBig(i int64) Big {
	return NewBigFromBigInt(big.NewInt(i))
}

func NewBigFromString(s string) (Big, error) {
	i, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return Big{}, errors.Errorf("not proper Big string, %q", s)
	}
	return NewBigFromBigInt(i), nil
}

func MustBigFromString(s string) Big {
	i, ok := new(big.Int).SetString(s, 10)
	if ok {
		return NewBigFromBigInt(i)
	}
	panic(errors.Errorf("not proper Big string, %q", s))
}

func NewBigFromInterface(a interface{}) (Big, error) {
	switch t := a.(type) {
	case int:
		return NewBig(int64(t)), nil
	case int8:
		return NewBig(int64(t)), nil
	case int32:
		return NewBig(int64(t)), nil
	case int64:
		return NewBig(t), nil
	case uint:
		return NewBig(int64(t)), nil
	case uint8:
		return NewBig(int64(t)), nil
	case uint32:
		return NewBig(int64(t)), nil
	case uint64:
		return NewBig(int64(t)), nil
	case string:
		n, err := NewBigFromString(t)
		if err != nil {
			return NilBig, errors.Errorf("invalid Big value, %q", t)
		}
		return n, nil
	default:
		return NilBig, errors.Errorf("unknown type of Big value, %T", a)
	}
}

func (a Big) String() string {
	if a.Int == nil {
		return NilBigString
	}
	return a.Int.String()
}

func (a Big) IsZero() bool {
	if a.Int == nil {
		return true
	}

	return a.Int.Cmp(ZeroBig.Int) == 0
}

func (a Big) OverZero() bool {
	if a.Int == nil {
		return false
	}

	return a.Int.Cmp(ZeroBig.Int) > 0
}

func (a Big) OverNil() bool {
	if a.Int == nil {
		return false
	}

	return a.Int.Cmp(ZeroBig.Int) >= 0
}

func (a Big) Equal(b Big) bool {
	if a.Int == nil {
		return false
	}

	return a.Int.Cmp(b.Int) == 0
}

func (a Big) Compare(b Big) int {
	if a.Int == nil {
		return -1
	}

	return a.Int.Cmp(b.Int)
}

func (Big) IsValid([]byte) error {
	return nil
}

func (a Big) Add(b Big) Big {
	return NewBigFromBigInt((new(big.Int)).Add(a.Int, b.Int))
}

func (a Big) Sub(b Big) Big {
	return NewBigFromBigInt((new(big.Int)).Sub(a.Int, b.Int))
}

func (a Big) Mul(b Big) Big {
	return NewBigFromBigInt((new(big.Int)).Mul(a.Int, b.Int))
}

func (a Big) MulInt64(b int64) Big {
	i := big.NewInt(b)
	return NewBigFromBigInt((new(big.Int)).Mul(a.Int, i))
}

func (a Big) MulFloat64(b float64) Big {
	af, _ := new(big.Float).SetString(a.Int.String())
	bf := big.NewFloat(b)

	c := new(big.Int)
	_, _ = new(big.Float).Mul(af, bf).Int(c)

	return NewBigFromBigInt(c)
}

func (a Big) Div(b Big) Big {
	return NewBigFromBigInt((new(big.Int)).Div(a.Int, b.Int))
}

func (a Big) Neg() Big {
	return NewBigFromBigInt((new(big.Int)).Neg(a.Int))
}
