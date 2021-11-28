package currency

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

type testKeyUpdaterOperation struct {
	baseTestOperationProcessor
}

func (t *testKeyUpdaterOperation) processor(cp *CurrencyPool, pool *storage.Statepool) prprocessor.OperationProcessor {
	copr, err := NewOperationProcessor(cp).
		SetProcessor(KeyUpdaterHinter, NewKeyUpdaterProcessor(cp))
	t.NoError(err)

	if pool == nil {
		return copr
	}

	return copr.New(pool)
}

func (t *testKeyUpdaterOperation) newOperation(target base.Address, keys Keys, pks []key.Privatekey, cid CurrencyID) KeyUpdater {
	token := util.UUID().Bytes()
	fact := NewKeyUpdaterFact(token, target, keys, cid)

	var fs []base.FactSign
	for _, pk := range pks {
		sig, err := base.NewFactSignature(pk, fact, nil)
		if err != nil {
			panic(err)
		}

		fs = append(fs, base.NewBaseFactSign(pk.Publickey(), sig))
	}

	op, err := NewKeyUpdater(fact, fs, "")
	if err != nil {
		panic(err)
	}

	err = op.IsValid(nil)
	if err != nil {
		panic(err)
	}

	return op
}

func (t *testKeyUpdaterOperation) TestNew() {
	am := NewAmount(NewBig(3), t.cid)
	sa, st := t.newAccount(true, []Amount{am})

	pool, _ := t.statepool(st)

	fee := NewBig(1)
	feeer := NewFixedFeeer(sa.Address, fee)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(t.cid, NewBig(99), NewTestAddress(), feeer)))

	opr := t.processor(cp, pool)

	npk := key.MustNewBTCPrivatekey()
	nkey, err := NewKey(npk.Publickey(), 100)
	t.NoError(err)
	nkeys, err := NewKeys([]Key{nkey}, 100)
	t.NoError(err)

	op := t.newOperation(sa.Address, nkeys, sa.Privs(), t.cid)

	t.NoError(opr.Process(op))

	// checking value
	var ns state.State
	var nb Amount
	for _, st := range pool.Updates() {
		if st.Key() == StateKeyAccount(sa.Address) {
			ns = st.GetState()
		} else if st.Key() == StateKeyBalance(sa.Address, am.Currency()) {
			i, err := StateBalanceValue(st.GetState())
			t.NoError(err)
			nb = i
		}
	}

	ac := ns.Value().Interface().(Account)
	ukeys := ac.Keys()
	t.True(nkeys.Equal(ukeys))

	t.True(am.Big().Sub(fee).Equal(nb.Big()))

	t.NoError(opr.Close())
}

func (t *testKeyUpdaterOperation) TestUnknownCurrency() {
	am := NewAmount(NewBig(3), CurrencyID("FINDME"))
	sa, st := t.newAccount(true, []Amount{am})

	pool, _ := t.statepool(st)
	feeer := NewFixedFeeer(sa.Address, NewBig(1))

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(t.cid, NewBig(99), NewTestAddress(), feeer)))

	opr := t.processor(cp, pool)

	npk := key.MustNewBTCPrivatekey()
	nkey, err := NewKey(npk.Publickey(), 100)
	t.NoError(err)
	nkeys, err := NewKeys([]Key{nkey}, 100)
	t.NoError(err)

	op := t.newOperation(sa.Address, nkeys, sa.Privs(), t.cid)

	err = opr.Process(op)

	var oper operation.ReasonError
	t.True(errors.As(err, &oper))
	t.Contains(err.Error(), "balance of target does not exist")
}

func (t *testKeyUpdaterOperation) TestEmptyBalance() {
	am := NewAmount(NewBig(0), t.cid)
	sa, st := t.newAccount(true, []Amount{am})

	pool, _ := t.statepool(st)
	feeer := NewFixedFeeer(sa.Address, NewBig(1))

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(t.cid, NewBig(99), NewTestAddress(), feeer)))

	opr := t.processor(cp, pool)

	npk := key.MustNewBTCPrivatekey()
	nkey, err := NewKey(npk.Publickey(), 100)
	t.NoError(err)
	nkeys, err := NewKeys([]Key{nkey}, 100)
	t.NoError(err)

	op := t.newOperation(sa.Address, nkeys, sa.Privs(), t.cid)

	err = opr.Process(op)

	var oper operation.ReasonError
	t.True(errors.As(err, &oper))
	t.Contains(err.Error(), "insufficient balance")
}

func (t *testKeyUpdaterOperation) TestTargetNotExist() {
	am := NewAmount(NewBig(3), t.cid)
	sa, _ := t.newAccount(false, []Amount{am})

	_, opr := t.statepool()
	_, err := opr.(*OperationProcessor).
		SetProcessor(KeyUpdaterHinter, NewKeyUpdaterProcessor(nil))
	t.NoError(err)

	npk := key.MustNewBTCPrivatekey()
	nkey, err := NewKey(npk.Publickey(), 100)
	t.NoError(err)
	nkeys, err := NewKeys([]Key{nkey}, 100)
	t.NoError(err)

	op := t.newOperation(sa.Address, nkeys, sa.Privs(), t.cid)

	err = opr.Process(op)

	var oper operation.ReasonError
	t.True(errors.As(err, &oper))
	t.Contains(err.Error(), "target keys does not exist")
}

func (t *testKeyUpdaterOperation) TestSameKeys() {
	am := NewAmount(NewBig(3), t.cid)
	sa, st := t.newAccount(true, []Amount{am})

	_, opr := t.statepool(st)
	_, err := opr.(*OperationProcessor).
		SetProcessor(KeyUpdaterHinter, NewKeyUpdaterProcessor(nil))
	t.NoError(err)

	op := t.newOperation(sa.Address, sa.Keys(), sa.Privs(), t.cid)

	err = opr.Process(op)

	var oper operation.ReasonError
	t.True(errors.As(err, &oper))
	t.Contains(err.Error(), "same Keys")
}

func (t *testKeyUpdaterOperation) TestWrongSigning() {
	am := NewAmount(NewBig(3), t.cid)
	sa, st := t.newAccount(true, []Amount{am})

	pool, _ := t.statepool(st)

	fee := NewBig(1)
	feeer := NewFixedFeeer(sa.Address, fee)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(t.cid, NewBig(99), NewTestAddress(), feeer)))

	opr := t.processor(cp, pool)

	npk := key.MustNewBTCPrivatekey()
	nkey, err := NewKey(npk.Publickey(), 100)
	t.NoError(err)
	nkeys, err := NewKeys([]Key{nkey}, 100)
	t.NoError(err)

	op := t.newOperation(sa.Address, nkeys, []key.Privatekey{key.MustNewBTCPrivatekey()}, t.cid)

	err = opr.Process(op)

	var oper operation.ReasonError
	t.True(errors.As(err, &oper))
	t.Contains(err.Error(), "invalid signing")
}

func TestKeyUpdaterOperation(t *testing.T) {
	suite.Run(t, new(testKeyUpdaterOperation))
}
