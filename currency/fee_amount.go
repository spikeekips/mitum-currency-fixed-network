package currency

import (
	"fmt"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
)

type FeeAmount interface {
	Min() Amount
	Fee(Amount) (Amount, error)
	Verbose() string
}

type NilFeeAmount struct {
}

func NewNilFeeAmount() NilFeeAmount {
	return NilFeeAmount{}
}

func (fa NilFeeAmount) Min() Amount {
	return ZeroAmount
}

func (fa NilFeeAmount) Fee(Amount) (Amount, error) {
	return ZeroAmount, nil
}

func (fa NilFeeAmount) Verbose() string {
	return `{"type": "nil", "amount": 0}`
}

type FixedFeeAmount struct {
	amount Amount
	isZero bool
}

func NewFixedFeeAmount(a Amount) FixedFeeAmount {
	return FixedFeeAmount{amount: a, isZero: a.IsZero()}
}

func (fa FixedFeeAmount) Min() Amount {
	return fa.amount
}

func (fa FixedFeeAmount) Fee(Amount) (Amount, error) {
	if fa.isZero {
		return ZeroAmount, nil
	}

	return fa.amount, nil
}

func (fa FixedFeeAmount) Verbose() string {
	return fmt.Sprintf(`{"type": "fixed", "amount": %q}`, fa.amount.String())
}

type RatioFeeAmount struct {
	ratio  float64 // 0 >=, or <= 1.0
	min    Amount
	isZero bool
	isOne  bool
}

func NewRatioFeeAmount(ratio float64, min Amount) (RatioFeeAmount, error) {
	if ratio < 0 || ratio > 1 {
		return RatioFeeAmount{}, xerrors.Errorf("invalid ratio, %v; it should be 0 >=, <= 1", ratio)
	}

	return RatioFeeAmount{ratio: ratio, min: min, isZero: ratio == 0, isOne: ratio == 1}, nil
}

func (fa RatioFeeAmount) Min() Amount {
	return fa.min
}

func (fa RatioFeeAmount) Fee(a Amount) (Amount, error) {
	if fa.isZero {
		return ZeroAmount, nil
	} else if a.IsZero() {
		return fa.min, nil
	}

	if fa.isOne {
		return a, nil
	} else if f := a.MulFloat64(fa.ratio); f.Compare(fa.min) < 0 {
		return fa.min, nil
	} else {
		return f, nil
	}
}

func (fa RatioFeeAmount) Verbose() string {
	return fmt.Sprintf(`{"type": "ratio", "ratio": %f, "min": %q}`, fa.ratio, fa.min.String())
}

func NewFeeToken(fa FeeAmount, height base.Height) []byte {
	return []byte(fmt.Sprintf("%s, height=%v", fa.Verbose(), height))
}
