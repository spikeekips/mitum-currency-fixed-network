package currency

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

const (
	FeeerNil   = "nil"
	FeeerFixed = "fixed"
	FeeerRatio = "ratio"
)

var (
	NilFeeerType   = hint.Type("mitum-currency-nil-feeer")
	NilFeeerHint   = hint.NewHint(NilFeeerType, "v0.0.1")
	FixedFeeerType = hint.Type("mitum-currency-fixed-feeer")
	FixedFeeerHint = hint.NewHint(FixedFeeerType, "v0.0.1")
	RatioFeeerType = hint.Type("mitum-currency-ratio-feeer")
	RatioFeeerHint = hint.NewHint(RatioFeeerType, "v0.0.1")
)

var UnlimitedMaxFeeAmount = NewBig(-1)

type Feeer interface {
	isvalid.IsValider
	hint.Hinter
	Type() string
	Bytes() []byte
	Receiver() base.Address
	Min() Big
	Fee(Big) (Big, error)
}

type NilFeeer struct{}

func NewNilFeeer() NilFeeer {
	return NilFeeer{}
}

func (NilFeeer) Type() string {
	return FeeerNil
}

func (NilFeeer) Hint() hint.Hint {
	return NilFeeerHint
}

func (NilFeeer) Bytes() []byte {
	return nil
}

func (NilFeeer) Receiver() base.Address {
	return nil
}

func (NilFeeer) Min() Big {
	return ZeroBig
}

func (NilFeeer) Fee(Big) (Big, error) {
	return ZeroBig, nil
}

func (NilFeeer) IsValid([]byte) error {
	return nil
}

type FixedFeeer struct {
	receiver base.Address
	amount   Big
}

func NewFixedFeeer(receiver base.Address, amount Big) FixedFeeer {
	return FixedFeeer{receiver: receiver, amount: amount}
}

func (FixedFeeer) Type() string {
	return FeeerFixed
}

func (FixedFeeer) Hint() hint.Hint {
	return FixedFeeerHint
}

func (fa FixedFeeer) Bytes() []byte {
	return util.ConcatBytesSlice(fa.receiver.Bytes(), fa.amount.Bytes())
}

func (fa FixedFeeer) Receiver() base.Address {
	return fa.receiver
}

func (fa FixedFeeer) Min() Big {
	return fa.amount
}

func (fa FixedFeeer) Fee(Big) (Big, error) {
	if fa.isZero() {
		return ZeroBig, nil
	}

	return fa.amount, nil
}

func (fa FixedFeeer) IsValid([]byte) error {
	if err := fa.receiver.IsValid(nil); err != nil {
		return errors.Wrap(err, "invalid receiver for fixed feeer")
	}

	if !fa.amount.OverNil() {
		return errors.Errorf("fixed feeer amount under zero")
	}

	return nil
}

func (fa FixedFeeer) isZero() bool {
	return fa.amount.IsZero()
}

type RatioFeeer struct {
	receiver base.Address
	ratio    float64 // 0 >=, or <= 1.0
	min      Big
	max      Big
}

func NewRatioFeeer(receiver base.Address, ratio float64, min, max Big) RatioFeeer {
	return RatioFeeer{
		receiver: receiver,
		ratio:    ratio,
		min:      min,
		max:      max,
	}
}

func (RatioFeeer) Type() string {
	return FeeerRatio
}

func (RatioFeeer) Hint() hint.Hint {
	return RatioFeeerHint
}

func (fa RatioFeeer) Bytes() []byte {
	var rb bytes.Buffer
	_ = binary.Write(&rb, binary.BigEndian, fa.ratio)

	return util.ConcatBytesSlice(fa.receiver.Bytes(), rb.Bytes(), fa.min.Bytes(), fa.max.Bytes())
}

func (fa RatioFeeer) Receiver() base.Address {
	return fa.receiver
}

func (fa RatioFeeer) Min() Big {
	return fa.min
}

func (fa RatioFeeer) Fee(a Big) (Big, error) {
	if fa.isZero() {
		return ZeroBig, nil
	} else if a.IsZero() {
		return fa.min, nil
	}

	if fa.isOne() {
		return a, nil
	} else if f := a.MulFloat64(fa.ratio); f.Compare(fa.min) < 0 {
		return fa.min, nil
	} else {
		if !fa.isUnlimited() && f.Compare(fa.max) > 0 {
			return fa.max, nil
		}
		return f, nil
	}
}

func (fa RatioFeeer) IsValid([]byte) error {
	if err := fa.receiver.IsValid(nil); err != nil {
		return errors.Wrap(err, "invalid receiver for ratio feeer")
	}

	if fa.ratio < 0 || fa.ratio > 1 {
		return errors.Errorf("invalid ratio, %v; it should be 0 >=, <= 1", fa.ratio)
	}

	if !fa.min.OverNil() {
		return errors.Errorf("ratio feeer min amount under zero")
	} else if !fa.max.Equal(UnlimitedMaxFeeAmount) {
		if !fa.max.OverNil() {
			return errors.Errorf("ratio feeer max amount under zero")
		} else if fa.min.Compare(fa.max) > 0 {
			return errors.Errorf("ratio feeer min should over max")
		}
	}

	return nil
}

func (fa RatioFeeer) isUnlimited() bool {
	return fa.max.Equal(UnlimitedMaxFeeAmount)
}

func (fa RatioFeeer) isZero() bool {
	return fa.ratio == 0
}

func (fa RatioFeeer) isOne() bool {
	return fa.ratio == 1
}

func NewFeeToken(feeer Feeer, height base.Height) []byte {
	return util.ConcatBytesSlice(feeer.Bytes(), height.Bytes())
}
