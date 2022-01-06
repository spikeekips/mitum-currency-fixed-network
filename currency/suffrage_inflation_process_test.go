package currency

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
)

type testSuffrageInflationOperations struct {
	baseTestOperationProcessor
	cid CurrencyID
}

func (t *testSuffrageInflationOperations) SetupSuite() {
	t.cid = CurrencyID("SHOWME")
}

func (t *testSuffrageInflationOperations) newOperation(keys []key.Privatekey, items []SuffrageInflationItem) SuffrageInflation {
	token := util.UUID().Bytes()
	fact := NewSuffrageInflationFact(token, items)

	var fs []base.FactSign
	for _, pk := range keys {
		sig, err := base.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, base.NewBaseFactSign(pk.Publickey(), sig))
	}

	tf, err := NewSuffrageInflation(fact, fs, "")
	t.NoError(err)

	t.NoError(tf.IsValid(nil))

	return tf
}

func (t *testSuffrageInflationOperations) processor(n int, pool *storage.Statepool) ([]key.Privatekey, prprocessor.OperationProcessor, *CurrencyPool) {
	privs := make([]key.Privatekey, n)
	for i := 0; i < n; i++ {
		privs[i] = key.NewBasePrivatekey()
	}

	pubs := make([]key.Publickey, len(privs))
	for i := range privs {
		pubs[i] = privs[i].Publickey()
	}
	threshold, err := base.NewThreshold(uint(len(privs)), 100)
	t.NoError(err)

	opr := NewOperationProcessor(nil)

	cp := NewCurrencyPool()
	_, err = opr.SetProcessor(SuffrageInflationHinter, NewSuffrageInflationProcessor(cp, pubs, threshold))
	t.NoError(err)

	return privs, opr.New(pool), cp
}

func (t *testSuffrageInflationOperations) TestNew() {
	cids := make([]CurrencyID, 3)
	for i := 0; i < 3; i++ {
		cids[i] = CurrencyID(fmt.Sprintf("XX%d", i))
	}

	sa, sts := t.newAccount(true, []Amount{NewAmount(NewBig(10), cids[0])})
	pool, _ := t.statepool(sts)
	privs, opr, cp := t.processor(2, pool)

	feeer := NewFixedFeeer(sa.Address, ZeroBig)
	for i := range cids {
		cd := t.newCurrencyDesignState(cids[i], NewBig(99), NewTestAddress(), feeer)
		t.NoError(cp.Set(cd))
	}

	items := make([]SuffrageInflationItem, 3)
	for i := 0; i < 3; i++ {
		items[i] = NewSuffrageInflationItem(sa.Address, NewAmount(NewBig(100), cids[i]))
	}

	op := t.newOperation(privs, items)
	t.NoError(op.IsValid(nil))

	t.NoError(opr.Process(op))
	t.NoError(opr.Close())

	tb := map[CurrencyID]Amount{}
	tcid := map[CurrencyID]CurrencyDesign{}

	for _, st := range pool.Updates() {
		switch {
		case IsStateBalanceKey(st.Key()):
			i, err := StateBalanceValue(st.GetState())
			t.NoError(err)

			tb[i.Currency()] = i
		case IsStateCurrencyDesignKey(st.Key()):
			i, err := StateCurrencyDesignValue(st.GetState())
			t.NoError(err)
			tcid[i.Currency()] = i
		}
	}

	t.True(tb[cids[0]].Big().Equal(NewBig(110)))
	t.True(tb[cids[1]].Big().Equal(NewBig(100)))
	t.True(tb[cids[2]].Big().Equal(NewBig(100)))

	t.True(tcid[cids[0]].Aggregate().Equal(NewBig(199)))
	t.True(tcid[cids[1]].Aggregate().Equal(NewBig(199)))
	t.True(tcid[cids[2]].Aggregate().Equal(NewBig(199)))
}

func (t *testSuffrageInflationOperations) TestUnknownReceiver() {
	cids := make([]CurrencyID, 3)
	for i := 0; i < 3; i++ {
		cids[i] = CurrencyID(fmt.Sprintf("XX%d", i))
	}

	pool, _ := t.statepool()
	privs, opr, cp := t.processor(2, pool)

	feeer := NewFixedFeeer(base.RandomStringAddress(), ZeroBig)
	for i := range cids {
		cd := t.newCurrencyDesignState(cids[i], NewBig(99), NewTestAddress(), feeer)
		t.NoError(cp.Set(cd))
	}

	items := make([]SuffrageInflationItem, 3)
	for i := 0; i < 3; i++ {
		items[i] = NewSuffrageInflationItem(base.RandomStringAddress(), NewAmount(NewBig(100), cids[i]))
	}

	op := t.newOperation(privs, items)
	t.NoError(op.IsValid(nil))

	err := opr.Process(op)
	var oper operation.ReasonError
	t.True(errors.As(err, &oper))
	t.Contains(err.Error(), "unknown receiver of SuffrageInflation")
}

func (t *testSuffrageInflationOperations) TestUnknownCurrency() {
	cids := make([]CurrencyID, 3)
	for i := 0; i < 3; i++ {
		cids[i] = CurrencyID(fmt.Sprintf("XX%d", i))
	}

	sa, sts := t.newAccount(true, []Amount{NewAmount(NewBig(10), cids[0])})
	pool, _ := t.statepool(sts)
	privs, opr, cp := t.processor(2, pool)

	feeer := NewFixedFeeer(sa.Address, ZeroBig)
	for i := range cids {
		cd := t.newCurrencyDesignState(cids[i], NewBig(99), NewTestAddress(), feeer)
		t.NoError(cp.Set(cd))
	}

	items := make([]SuffrageInflationItem, 3)
	for i := 0; i < 3; i++ {
		cid := cids[i]
		if i == len(cids)-1 {
			cid = CurrencyID("FINEME")
		}

		items[i] = NewSuffrageInflationItem(sa.Address, NewAmount(NewBig(100), cid))
	}

	op := t.newOperation(privs, items)
	t.NoError(op.IsValid(nil))

	err := opr.Process(op)

	var oper operation.ReasonError
	t.True(errors.As(err, &oper))
	t.Contains(err.Error(), "unknown currency")
}

func TestSuffrageInflationOperations(t *testing.T) {
	suite.Run(t, new(testSuffrageInflationOperations))
}
