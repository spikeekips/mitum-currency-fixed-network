package currency

import (
	"time"

	"github.com/pkg/errors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	FeeOperationFactType   = hint.Type("mitum-currency-fee-operation-fact")
	FeeOperationFactHint   = hint.NewHint(FeeOperationFactType, "v0.0.1")
	FeeOperationFactHinter = FeeOperationFact{BaseHinter: hint.NewBaseHinter(FeeOperationFactHint)}
	FeeOperationType       = hint.Type("mitum-currency-fee-operation")
	FeeOperationHint       = hint.NewHint(FeeOperationType, "v0.0.1")
	FeeOperationHinter     = FeeOperation{BaseHinter: hint.NewBaseHinter(FeeOperationHint)}
)

type FeeOperationFact struct {
	hint.BaseHinter
	h       valuehash.Hash
	token   []byte
	amounts []Amount
}

func NewFeeOperationFact(height base.Height, ams map[CurrencyID]Big) FeeOperationFact {
	amounts := make([]Amount, len(ams))
	var i int
	for cid := range ams {
		amounts[i] = NewAmount(ams[cid], cid)
		i++
	}

	// TODO replace random bytes with height
	fact := FeeOperationFact{
		BaseHinter: hint.NewBaseHinter(FeeOperationFactHint),
		token:      height.Bytes(), // for unique token
		amounts:    amounts,
	}
	fact.h = valuehash.NewSHA256(fact.Bytes())

	return fact
}

func (fact FeeOperationFact) Hash() valuehash.Hash {
	return fact.h
}

func (fact FeeOperationFact) Bytes() []byte {
	bs := make([][]byte, len(fact.amounts)+1)
	bs[0] = fact.token

	for i := range fact.amounts {
		bs[i+1] = fact.amounts[i].Bytes()
	}

	return util.ConcatBytesSlice(bs...)
}

func (fact FeeOperationFact) IsValid([]byte) error {
	if len(fact.token) < 1 {
		return isvalid.InvalidError.Errorf("empty token for FeeOperationFact")
	}

	if err := isvalid.Check(nil, false, fact.h); err != nil {
		return err
	}

	for i := range fact.amounts {
		if err := fact.amounts[i].IsValid(nil); err != nil {
			return err
		}
	}

	return nil
}

func (fact FeeOperationFact) Token() []byte {
	return fact.token
}

func (fact FeeOperationFact) Amounts() []Amount {
	return fact.amounts
}

type FeeOperation struct {
	hint.BaseHinter
	fact FeeOperationFact
	h    valuehash.Hash
}

func NewFeeOperation(fact FeeOperationFact) FeeOperation {
	op := FeeOperation{BaseHinter: hint.NewBaseHinter(FeeOperationHint), fact: fact}
	op.h = op.GenerateHash()

	return op
}

func (op FeeOperation) Fact() base.Fact {
	return op.fact
}

func (op FeeOperation) Hash() valuehash.Hash {
	return op.h
}

func (FeeOperation) Signs() []base.FactSign {
	return nil
}

func (op FeeOperation) IsValid([]byte) error {
	if err := isvalid.Check(nil, false, op.BaseHinter, op.h); err != nil {
		return err
	}

	if l := len(op.fact.Token()); l < 1 {
		return isvalid.InvalidError.Errorf("FeeOperation has empty token")
	} else if l > operation.MaxTokenSize {
		return isvalid.InvalidError.Errorf("FeeOperation token size too large: %d > %d", l, operation.MaxTokenSize)
	}

	if err := op.fact.IsValid(nil); err != nil {
		return err
	}

	if !op.Hash().Equal(op.GenerateHash()) {
		return isvalid.InvalidError.Errorf("wrong FeeOperation hash")
	}

	return nil
}

func (op FeeOperation) GenerateHash() valuehash.Hash {
	return valuehash.NewSHA256(op.Fact().Hash().Bytes())
}

func (FeeOperation) AddFactSigns(...base.FactSign) (base.FactSignUpdater, error) {
	return nil, nil
}

func (FeeOperation) LastSignedAt() time.Time {
	return time.Time{}
}

func (FeeOperation) Process(
	func(key string) (state.State, bool, error),
	func(valuehash.Hash, ...state.State) error,
) error {
	return nil
}

type FeeOperationProcessor struct {
	FeeOperation
	cp *CurrencyPool
}

func NewFeeOperationProcessor(cp *CurrencyPool, op FeeOperation) state.Processor {
	return &FeeOperationProcessor{
		cp:           cp,
		FeeOperation: op,
	}
}

func (opp *FeeOperationProcessor) Process(
	getState func(key string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) error {
	fact := opp.Fact().(FeeOperationFact)

	sts := make([]state.State, len(fact.amounts))
	for i := range fact.amounts {
		am := fact.amounts[i]
		var feeer Feeer
		j, found := opp.cp.Feeer(am.Currency())
		if !found {
			return errors.Errorf("unknown currency id, %q found for FeeOperation", am.Currency())
		}
		feeer = j

		if feeer.Receiver() == nil {
			continue
		}

		if err := checkExistsState(StateKeyAccount(feeer.Receiver()), getState); err != nil {
			return err
		} else if st, _, err := getState(StateKeyBalance(feeer.Receiver(), am.Currency())); err != nil {
			return err
		} else {
			rb := NewAmountState(st, am.Currency())

			sts[i] = rb.Add(am.Big())
		}
	}

	return setState(fact.Hash(), sts...)
}
