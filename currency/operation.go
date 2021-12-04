package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

type BaseOperation struct {
	operation.BaseOperation
	Memo string
}

func NewBaseOperationFromFact(
	ht hint.Hint, fact operation.OperationFact, fs []base.FactSign, memo string,
) (BaseOperation, error) {
	bo, err := operation.NewBaseOperationFromFact(ht, fact, fs)
	if err != nil {
		return BaseOperation{}, err
	}
	op := BaseOperation{BaseOperation: bo, Memo: memo}

	op.BaseOperation = bo.SetHash(op.GenerateHash())

	return op, nil
}

func (op BaseOperation) IsValid(networkID []byte) error {
	if err := op.BaseOperation.BaseHinter.IsValid(nil); err != nil {
		return err
	}

	if err := IsValidMemo(op.Memo); err != nil {
		return err
	}

	return operation.IsValidOperation(op, networkID)
}

func (op BaseOperation) GenerateHash() valuehash.Hash {
	bs := make([][]byte, len(op.Signs())+1)
	for i := range op.Signs() {
		bs[i] = op.Signs()[i].Bytes()
	}

	bs[len(bs)-1] = []byte(op.Memo)

	e := util.ConcatBytesSlice(op.Fact().Hash().Bytes(), util.ConcatBytesSlice(bs...))

	return valuehash.NewSHA256(e)
}

func (op BaseOperation) AddFactSigns(fs ...base.FactSign) (base.FactSignUpdater, error) {
	o, err := op.BaseOperation.AddFactSigns(fs...)
	if err != nil {
		return nil, err
	}
	op.BaseOperation = o.(operation.BaseOperation)

	op.BaseOperation = op.SetHash(op.GenerateHash())

	return op, nil
}

func operationHinter(ht hint.Hint) BaseOperation {
	return BaseOperation{BaseOperation: operation.EmptyBaseOperation(ht)}
}

func IsValidOperationFact(fact operation.OperationFact, b []byte) error {
	if err := operation.IsValidOperationFact(fact, b); err != nil {
		return err
	}

	hg, ok := fact.(valuehash.HashGenerator)
	if !ok {
		return nil
	}

	if !fact.Hash().Equal(hg.GenerateHash()) {
		return isvalid.InvalidError.Errorf("wrong Fact hash")
	}

	return nil
}
