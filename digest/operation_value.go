package digest

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util/hint"
)

var (
	OperationValueType = hint.MustNewType(0xa0, 0x19, "mitum-currency-operation-value")
	OperationValueHint = hint.MustHint(OperationValueType, "0.0.1")
)

type OperationValue struct {
	op          operation.Operation
	height      base.Height
	confirmedAt time.Time
	inStates    bool
}

func NewOperationValue(
	op operation.Operation,
	height base.Height,
	confirmedAt time.Time,
	inStates bool,
) OperationValue {
	return OperationValue{op: op, height: height, confirmedAt: confirmedAt, inStates: inStates}
}

func (va OperationValue) Hint() hint.Hint {
	return OperationValueHint
}

func (va OperationValue) Operation() operation.Operation {
	return va.op
}

func (va OperationValue) Height() base.Height {
	return va.height
}

func (va OperationValue) ConfirmedAt() time.Time {
	return va.confirmedAt
}

func (va OperationValue) InState() bool {
	return va.inStates
}
