package digest

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util/hint"
)

var (
	OperationValueType = hint.Type("mitum-currency-operation-value")
	OperationValueHint = hint.NewHint(OperationValueType, "v0.0.1")
)

type OperationValue struct {
	op          operation.Operation
	height      base.Height
	confirmedAt time.Time
	inState     bool
	reason      operation.ReasonError
	index       uint64
}

func NewOperationValue(
	op operation.Operation,
	height base.Height,
	confirmedAt time.Time,
	inState bool,
	reason operation.ReasonError,
	index uint64,
) OperationValue {
	return OperationValue{
		op:          op,
		height:      height,
		confirmedAt: confirmedAt,
		inState:     inState,
		reason:      reason,
		index:       index,
	}
}

func (OperationValue) Hint() hint.Hint {
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
	return va.inState
}

func (va OperationValue) Reason() operation.ReasonError {
	return va.reason
}

// Index indicates the index number of Operation in OperationTree of block.
func (va OperationValue) Index() uint64 {
	return va.index
}
