package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	CurrencyPolicyUpdaterFactType   = hint.Type("mitum-currency-currency-policy-updater-operation-fact")
	CurrencyPolicyUpdaterFactHint   = hint.NewHint(CurrencyPolicyUpdaterFactType, "v0.0.1")
	CurrencyPolicyUpdaterFactHinter = CurrencyPolicyUpdaterFact{
		BaseHinter: hint.NewBaseHinter(CurrencyPolicyUpdaterFactHint),
	}
	CurrencyPolicyUpdaterType   = hint.Type("mitum-currency-currency-policy-updater-operation")
	CurrencyPolicyUpdaterHint   = hint.NewHint(CurrencyPolicyUpdaterType, "v0.0.1")
	CurrencyPolicyUpdaterHinter = CurrencyPolicyUpdater{BaseOperation: operationHinter(CurrencyPolicyUpdaterHint)}
)

type CurrencyPolicyUpdaterFact struct {
	hint.BaseHinter
	h      valuehash.Hash
	token  []byte
	cid    CurrencyID
	policy CurrencyPolicy
}

func NewCurrencyPolicyUpdaterFact(token []byte, cid CurrencyID, policy CurrencyPolicy) CurrencyPolicyUpdaterFact {
	fact := CurrencyPolicyUpdaterFact{
		BaseHinter: hint.NewBaseHinter(CurrencyPolicyUpdaterFactHint),
		token:      token,
		cid:        cid,
		policy:     policy,
	}

	fact.h = fact.GenerateHash()

	return fact
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

func (fact CurrencyPolicyUpdaterFact) IsValid(b []byte) error {
	if err := IsValidOperationFact(fact, b); err != nil {
		return err
	}

	if err := isvalid.Check(nil, false, fact.cid, fact.policy); err != nil {
		return isvalid.InvalidError.Errorf("invalid fact: %w", err)
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
	BaseOperation
}

func NewCurrencyPolicyUpdater(
	fact CurrencyPolicyUpdaterFact,
	fs []base.FactSign,
	memo string,
) (CurrencyPolicyUpdater, error) {
	bo, err := NewBaseOperationFromFact(CurrencyPolicyUpdaterHint, fact, fs, memo)
	if err != nil {
		return CurrencyPolicyUpdater{}, err
	}

	return CurrencyPolicyUpdater{BaseOperation: bo}, nil
}
