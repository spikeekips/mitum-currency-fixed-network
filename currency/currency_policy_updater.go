package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	CurrencyPolicyUpdaterFactType = hint.MustNewType(0xa0, 0x34, "mitum-currency-currency-policy-updater-operation-fact")
	CurrencyPolicyUpdaterFactHint = hint.MustHint(CurrencyPolicyUpdaterFactType, "0.0.1")
	CurrencyPolicyUpdaterType     = hint.MustNewType(0xa0, 0x35, "mitum-currency-currency-policy-updater-operation")
	CurrencyPolicyUpdaterHint     = hint.MustHint(CurrencyPolicyUpdaterType, "0.0.1")
)

type CurrencyPolicyUpdaterFact struct {
	h      valuehash.Hash
	token  []byte
	cid    CurrencyID
	policy CurrencyPolicy
}

func NewCurrencyPolicyUpdaterFact(token []byte, cid CurrencyID, policy CurrencyPolicy) CurrencyPolicyUpdaterFact {
	fact := CurrencyPolicyUpdaterFact{
		token:  token,
		cid:    cid,
		policy: policy,
	}

	fact.h = fact.GenerateHash()

	return fact
}

func (fact CurrencyPolicyUpdaterFact) Hint() hint.Hint {
	return CurrencyPolicyUpdaterFactHint
}

func (fact CurrencyPolicyUpdaterFact) Hash() valuehash.Hash {
	return fact.h
}

func (fact CurrencyPolicyUpdaterFact) Bytes() []byte {
	return util.ConcatBytesSlice(
		fact.token,
		fact.cid.Bytes(),
		fact.policy.Bytes(),
	)
}

func (fact CurrencyPolicyUpdaterFact) IsValid([]byte) error {
	if len(fact.token) < 1 {
		return xerrors.Errorf("empty token for CurrencyPolicyUpdaterFact")
	}

	if err := isvalid.Check([]isvalid.IsValider{
		fact.h,
		fact.cid,
		fact.policy,
	}, nil, false); err != nil {
		return xerrors.Errorf("invalid fact: %w", err)
	}

	if !fact.h.Equal(fact.GenerateHash()) {
		return isvalid.InvalidError.Errorf("wrong Fact hash")
	}

	return nil
}

func (fact CurrencyPolicyUpdaterFact) GenerateHash() valuehash.Hash {
	return valuehash.NewSHA256(fact.Bytes())
}

func (fact CurrencyPolicyUpdaterFact) Token() []byte {
	return fact.token
}

func (fact CurrencyPolicyUpdaterFact) Currency() CurrencyID {
	return fact.cid
}

func (fact CurrencyPolicyUpdaterFact) Policy() CurrencyPolicy {
	return fact.policy
}

type CurrencyPolicyUpdater struct {
	operation.BaseOperation
	Memo string
}

func NewCurrencyPolicyUpdater(
	fact CurrencyPolicyUpdaterFact,
	fs []operation.FactSign,
	memo string,
) (CurrencyPolicyUpdater, error) {
	if bo, err := operation.NewBaseOperationFromFact(CurrencyPolicyUpdaterHint, fact, fs); err != nil {
		return CurrencyPolicyUpdater{}, err
	} else {
		op := CurrencyPolicyUpdater{BaseOperation: bo, Memo: memo}

		op.BaseOperation = bo.SetHash(op.GenerateHash())

		return op, nil
	}
}

func (op CurrencyPolicyUpdater) Hint() hint.Hint {
	return CurrencyPolicyUpdaterHint
}

func (op CurrencyPolicyUpdater) IsValid(networkID []byte) error {
	if err := IsValidMemo(op.Memo); err != nil {
		return err
	}

	return operation.IsValidOperation(op, networkID)
}
