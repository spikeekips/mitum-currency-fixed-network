package currency

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
)

type testCurrencyPolicyUpdaterOperations struct {
	baseTestOperationProcessor
	cid CurrencyID
}

func (t *testCurrencyPolicyUpdaterOperations) SetupSuite() {
	t.cid = CurrencyID("SHOWME")
}

func (t *testCurrencyPolicyUpdaterOperations) newOperation(keys []key.Privatekey, cid CurrencyID, po CurrencyPolicy) CurrencyPolicyUpdater {
	token := util.UUID().Bytes()
	fact := NewCurrencyPolicyUpdaterFact(token, cid, po)

	var fs []base.FactSign
	for _, pk := range keys {
		sig, err := base.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, base.NewBaseFactSign(pk.Publickey(), sig))
	}

	tf, err := NewCurrencyPolicyUpdater(fact, fs, "")
	t.NoError(err)

	t.NoError(tf.IsValid(nil))

	return tf
}

func (t *testCurrencyPolicyUpdaterOperations) processor(n int) ([]key.Privatekey, *OperationProcessor) {
	privs := make([]key.Privatekey, n)
	for i := 0; i < n; i++ {
		privs[i] = key.MustNewBTCPrivatekey()
	}

	pubs := make([]key.Publickey, len(privs))
	for i := range privs {
		pubs[i] = privs[i].Publickey()
	}
	threshold, err := base.NewThreshold(uint(len(privs)), 100)
	t.NoError(err)

	cp := NewCurrencyPool()
	opr := NewOperationProcessor(cp)
	_, err = opr.SetProcessor(CurrencyPolicyUpdaterHinter, NewCurrencyPolicyUpdaterProcessor(cp, pubs, threshold))
	t.NoError(err)

	return privs, opr
}

func (t *testCurrencyPolicyUpdaterOperations) currencyDesign(big Big, cid CurrencyID, ga base.Address) CurrencyDesign {
	return NewCurrencyDesign(NewAmount(big, cid), ga, NewCurrencyPolicy(ZeroBig, NewNilFeeer()))
}

func (t *testCurrencyPolicyUpdaterOperations) TestNew() {
	var sts []state.State

	privs, copr := t.processor(3)

	ga, s := t.newAccount(true, []Amount{NewAmount(NewBig(10), t.cid)})
	sts = append(sts, s...)

	de := t.currencyDesign(NewBig(33), t.cid, ga.Address)

	{
		st, err := state.NewStateV0(StateKeyCurrencyDesign(de.Currency()), nil, base.Height(33))
		t.NoError(err)

		nst, err := SetStateCurrencyDesignValue(st, de)
		t.NoError(err)
		sts = append(sts, nst)

		t.NoError(copr.cp.Set(nst))
	}

	pool, _ := t.statepool(sts)

	opr := copr.New(pool)

	po := NewCurrencyPolicy(NewBig(1), NewFixedFeeer(ga.Address, NewBig(44)))
	op := t.newOperation(privs, t.cid, po)
	t.NoError(opr.Process(op))

	var ude CurrencyDesign
	for _, st := range pool.Updates() {
		switch st.Key() {
		case StateKeyCurrencyDesign(t.cid):
			i, err := StateCurrencyDesignValue(st.GetState())
			t.NoError(err)

			ude = i
		}
	}

	t.True(de.Amount.Equal(ude.Amount))
	t.NotEqual(de.Policy(), ude.Policy())
}

func (t *testCurrencyPolicyUpdaterOperations) TestEmptyPubs() {
	var sts []state.State

	ga, s := t.newAccount(true, []Amount{NewAmount(NewBig(10), t.cid)})
	sts = append(sts, s...)

	de := t.currencyDesign(NewBig(33), t.cid, ga.Address)

	{
		st, err := state.NewStateV0(StateKeyCurrencyDesign(de.Currency()), nil, base.Height(33))
		t.NoError(err)

		nst, err := SetStateCurrencyDesignValue(st, de)
		t.NoError(err)
		sts = append(sts, nst)
	}

	pool, _ := t.statepool(sts)

	copr := NewOperationProcessor(nil)
	_, err := copr.SetProcessor(CurrencyPolicyUpdaterHinter, func(op state.Processor) (state.Processor, error) {
		if i, ok := op.(CurrencyPolicyUpdater); !ok {
			return nil, errors.Errorf("not CurrencyPolicyUpdater, %T", op)
		} else {
			return &CurrencyPolicyUpdaterProcessor{
				CurrencyPolicyUpdater: i,
			}, nil
		}
	})
	t.NoError(err)

	opr := copr.New(pool)

	po := NewCurrencyPolicy(NewBig(44), NewFixedFeeer(ga.Address, NewBig(44)))
	op := t.newOperation(ga.Privs(), t.cid, po)

	err = opr.Process(op)
	var oper operation.ReasonError
	t.True(errors.As(err, &oper))
	t.Contains(err.Error(), "empty publickeys")
}

func (t *testCurrencyPolicyUpdaterOperations) TestNotEnoughSigns() {
	var sts []state.State

	privs, copr := t.processor(3)

	ga, s := t.newAccount(true, []Amount{NewAmount(NewBig(10), t.cid)})
	sts = append(sts, s...)

	de := t.currencyDesign(NewBig(33), t.cid, ga.Address)

	{
		st, err := state.NewStateV0(StateKeyCurrencyDesign(de.Currency()), nil, base.Height(33))
		t.NoError(err)

		nst, err := SetStateCurrencyDesignValue(st, de)
		t.NoError(err)
		sts = append(sts, nst)
	}

	pool, _ := t.statepool(sts)

	opr := copr.New(pool)

	po := NewCurrencyPolicy(NewBig(44), NewFixedFeeer(ga.Address, NewBig(44)))
	op := t.newOperation(privs[:2], t.cid, po)

	err := opr.Process(op)
	var oper operation.ReasonError
	t.True(errors.As(err, &oper))
	t.Contains(err.Error(), "not enough suffrage signs")
}

func (t *testCurrencyPolicyUpdaterOperations) TestUnknownCurrency() {
	var sts []state.State

	privs, copr := t.processor(3)

	ga, s := t.newAccount(true, []Amount{NewAmount(NewBig(10), t.cid)})
	sts = append(sts, s...)

	de := t.currencyDesign(NewBig(33), t.cid, ga.Address)

	{
		st, err := state.NewStateV0(StateKeyCurrencyDesign(de.Currency()), nil, base.Height(33))
		t.NoError(err)

		nst, err := SetStateCurrencyDesignValue(st, de)
		t.NoError(err)
		sts = append(sts, nst)

		t.NoError(copr.cp.Set(nst))
	}

	pool, _ := t.statepool(sts)

	opr := copr.New(pool)

	po := NewCurrencyPolicy(NewBig(1), NewFixedFeeer(ga.Address, NewBig(44)))
	op := t.newOperation(privs, "FINEME", po)

	err := opr.Process(op)

	var oper operation.ReasonError
	t.True(errors.As(err, &oper))
	t.Contains(err.Error(), "unknown currency")
}

func (t *testCurrencyPolicyUpdaterOperations) TestUnknownReceiver() {
	var sts []state.State

	privs, copr := t.processor(3)

	ga, s := t.newAccount(true, []Amount{NewAmount(NewBig(10), t.cid)})
	sts = append(sts, s...)

	de := t.currencyDesign(NewBig(33), t.cid, ga.Address)

	{
		st, err := state.NewStateV0(StateKeyCurrencyDesign(de.Currency()), nil, base.Height(33))
		t.NoError(err)

		nst, err := SetStateCurrencyDesignValue(st, de)
		t.NoError(err)
		sts = append(sts, nst)

		t.NoError(copr.cp.Set(nst))
	}

	pool, _ := t.statepool(sts)

	opr := copr.New(pool)

	po := NewCurrencyPolicy(NewBig(1), NewFixedFeeer(base.RandomStringAddress(), NewBig(44)))
	op := t.newOperation(privs, t.cid, po)

	err := opr.Process(op)

	var oper operation.ReasonError
	t.True(errors.As(err, &oper))
	t.Contains(err.Error(), "feeer receiver account not found")
}

func TestCurrencyPolicyUpdaterOperations(t *testing.T) {
	suite.Run(t, new(testCurrencyPolicyUpdaterOperations))
}
