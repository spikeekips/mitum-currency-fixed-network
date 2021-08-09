package currency

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/tree"
)

type testTransfersOperations struct {
	baseTestOperationProcessor
	cid CurrencyID
}

func (t *testTransfersOperations) SetupSuite() {
	t.cid = CurrencyID("SHOWME")
}

func (t *testTransfersOperations) processor(cp *CurrencyPool, pool *storage.Statepool) prprocessor.OperationProcessor {
	copr, err := NewOperationProcessor(cp).
		SetProcessor(Transfers{}, NewTransfersProcessor(cp))
	t.NoError(err)

	if pool == nil {
		return copr
	}

	return copr.New(pool)
}

func (t *testTransfersOperations) newTransfersItem(receiver base.Address, big Big) TransfersItem {
	am := []Amount{NewAmount(big, t.cid)}

	return NewTransfersItemMultiAmounts(receiver, am)
}

func (t *testTransfersOperations) newTransfer(sender base.Address, keys []key.Privatekey, items []TransfersItem) Transfers {
	token := util.UUID().Bytes()
	fact := NewTransfersFact(token, sender, items)

	var fs []operation.FactSign
	for _, pk := range keys {
		sig, err := operation.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
	}

	tf, err := NewTransfers(fact, fs, "")
	t.NoError(err)

	t.NoError(tf.IsValid(nil))

	return tf
}

func (t *testTransfersOperations) TestSenderNotExist() {
	sa, _ := t.newAccount(false, []Amount{NewAmount(NewBig(10), t.cid)})
	ra, _ := t.newAccount(false, nil)

	items := []TransfersItem{t.newTransfersItem(ra.Address, NewBig(10))}
	tf := t.newTransfer(sa.Address, sa.Privs(), items)

	pool, _ := t.statepool()
	feeer := NewFixedFeeer(sa.Address, ZeroBig)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(t.cid, NewBig(99), NewTestAddress(), feeer)))

	opr := t.processor(cp, pool)

	err := opr.Process(tf)

	var oper operation.ReasonError
	t.True(errors.As(err, &oper))
	t.Contains(err.Error(), "does not exist")
}

func (t *testTransfersOperations) TestReceiverNotExist() {
	sa, sts := t.newAccount(true, []Amount{NewAmount(NewBig(10), t.cid)})
	ra, _ := t.newAccount(false, nil)

	pool, _ := t.statepool(sts)
	feeer := NewFixedFeeer(sa.Address, ZeroBig)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(t.cid, NewBig(99), NewTestAddress(), feeer)))

	opr := t.processor(cp, pool)

	items := []TransfersItem{t.newTransfersItem(ra.Address, NewBig(3))}
	tf := t.newTransfer(sa.Address, sa.Privs(), items)

	err := opr.Process(tf)

	var oper operation.ReasonError
	t.True(errors.As(err, &oper))
	t.Contains(err.Error(), "receiver does not exist")
}

func (t *testTransfersOperations) TestInsufficientBalance() {
	sa, st0 := t.newAccount(true, []Amount{NewAmount(NewBig(10), t.cid)})
	ra, st1 := t.newAccount(true, []Amount{NewAmount(NewBig(1), t.cid)})

	pool, _ := t.statepool(st0, st1)
	feeer := NewFixedFeeer(sa.Address, ZeroBig)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(t.cid, NewBig(99), NewTestAddress(), feeer)))

	opr := t.processor(cp, pool)

	items := []TransfersItem{t.newTransfersItem(ra.Address, NewBig(11))}
	tf := t.newTransfer(sa.Address, sa.Privs(), items)

	err := opr.Process(tf)

	var oper operation.ReasonError
	t.True(errors.As(err, &oper))
	t.Contains(err.Error(), "insufficient balance")
}

func (t *testTransfersOperations) TestSufficientBalance() {
	faBalance := NewAmount(NewBig(22), t.cid)
	saBalance := NewAmount(NewBig(33), t.cid)
	raBalance := NewAmount(NewBig(1), t.cid)
	fa, st0 := t.newAccount(true, []Amount{faBalance})
	sa, st1 := t.newAccount(true, []Amount{saBalance})
	ra, st2 := t.newAccount(true, []Amount{raBalance})

	pool, _ := t.statepool(st0, st1, st2)

	fee := NewBig(2)
	feeer := NewFixedFeeer(fa.Address, fee)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(t.cid, NewBig(99), NewTestAddress(), feeer)))

	opr := t.processor(cp, pool)

	sent := saBalance.Big().Sub(NewBig(10))

	tf := t.newTransfer(sa.Address, sa.Privs(), []TransfersItem{t.newTransfersItem(ra.Address, sent)})

	t.NoError(opr.Process(tf))
	t.NoError(opr.Close())

	var sst, rst, fst state.State
	for _, st := range pool.Updates() {
		switch st.Key() {
		case StateKeyBalance(sa.Address, t.cid):
			sst = st.GetState()
		case StateKeyBalance(ra.Address, t.cid):
			rst = st.GetState()
		case StateKeyBalance(fa.Address, t.cid):
			fst = st.GetState()
		}
	}

	// checking value
	sstv, _ := StateBalanceValue(sst)
	t.True(sstv.Big().Equal(saBalance.Big().Sub(sent).Sub(fee)))

	rstv, _ := StateBalanceValue(rst)
	t.True(rstv.Big().Equal(raBalance.Big().Add(sent)))

	fstv, _ := StateBalanceValue(fst)
	t.True(fstv.Big().Equal(faBalance.Big().Add(fee)))

	// check fee operation

	t.True(len(pool.AddedOperations()) > 0)
	var fo FeeOperation
	for _, op := range pool.AddedOperations() {
		if err := op.Hint().IsCompatible(FeeOperationHint); err == nil {
			fo = op.(FeeOperation)
		}
	}

	fof := fo.Fact().(FeeOperationFact)
	t.Equal(fee, fof.Amounts()[0].Big())
}

func (t *testTransfersOperations) TestMultipleItemsWithFee() {
	saBalance := NewAmount(NewBig(33), t.cid)
	sa, st0 := t.newAccount(true, []Amount{saBalance})
	ra0, rst0 := t.newAccount(true, []Amount{NewAmount(NewBig(0), t.cid)})
	ra1, rst1 := t.newAccount(true, []Amount{NewAmount(NewBig(0), t.cid)})

	pool, _ := t.statepool(st0, rst0, rst1)

	fee := NewBig(2)
	feeer := NewFixedFeeer(sa.Address, fee)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(t.cid, NewBig(99), NewTestAddress(), feeer)))

	opr := t.processor(cp, pool)

	sent := NewBig(10)

	token := util.UUID().Bytes()
	items := []TransfersItem{
		t.newTransfersItem(ra0.Address, sent),
		t.newTransfersItem(ra1.Address, sent),
	}
	fact := NewTransfersFact(token, sa.Address, items)
	sig, err := operation.NewFactSignature(sa.Privs()[0], fact, nil)
	t.NoError(err)
	fs := []operation.FactSign{operation.NewBaseFactSign(sa.Privs()[0].Publickey(), sig)}
	tf, err := NewTransfers(fact, fs, "")
	t.NoError(err)

	err = opr.Process(tf)
	t.NoError(err)

	var nst state.State
	var nam, nram0, nram1 Amount
	for _, st := range pool.Updates() {
		if st.Key() == StateKeyBalance(sa.Address, t.cid) {
			nst = st.GetState()
			nam, _ = StateBalanceValue(nst)
		} else if st.Key() == StateKeyBalance(ra0.Address, t.cid) {
			nram0, _ = StateBalanceValue(st.GetState())
		} else if st.Key() == StateKeyBalance(ra1.Address, t.cid) {
			nram1, _ = StateBalanceValue(st.GetState())
		}
	}

	t.Equal(saBalance.Big().Sub(sent.MulInt64(2)).Sub(fee.MulInt64(2)), nam.Big())
	t.Equal(sent, nram0.Big())
	t.Equal(sent, nram1.Big())
	t.Equal(fee.MulInt64(2), nst.(AmountState).Fee())
}

func (t *testTransfersOperations) TestInsufficientMultipleItemsWithFee() {
	saBalance := NewAmount(NewBig(33), t.cid)
	sa, st0 := t.newAccount(true, []Amount{saBalance})
	ra0, rst0 := t.newAccount(true, []Amount{NewAmount(NewBig(0), t.cid)})
	ra1, rst1 := t.newAccount(true, []Amount{NewAmount(NewBig(0), t.cid)})

	pool, _ := t.statepool(st0, rst0, rst1)

	fee := NewBig(2)
	feeer := NewFixedFeeer(sa.Address, fee)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(t.cid, NewBig(99), NewTestAddress(), feeer)))

	opr := t.processor(cp, pool)

	sent := NewBig(15)

	token := util.UUID().Bytes()
	items := []TransfersItem{
		t.newTransfersItem(ra0.Address, sent),
		t.newTransfersItem(ra1.Address, sent),
	}
	fact := NewTransfersFact(token, sa.Address, items)
	sig, err := operation.NewFactSignature(sa.Privs()[0], fact, nil)
	t.NoError(err)
	fs := []operation.FactSign{operation.NewBaseFactSign(sa.Privs()[0].Publickey(), sig)}
	tf, err := NewTransfers(fact, fs, "")
	t.NoError(err)

	err = opr.Process(tf)

	var oper operation.ReasonError
	t.True(errors.As(err, &oper))
	t.Contains(err.Error(), "insufficient balance")
}

func (t *testTransfersOperations) TestInSufficientBalanceWithFee() {
	saBalance := NewAmount(NewBig(33), t.cid)
	sa, st0 := t.newAccount(true, []Amount{saBalance})
	ra, st1 := t.newAccount(true, []Amount{NewAmount(NewBig(1), t.cid)})

	pool, _ := t.statepool(st0, st1)

	fee := NewBig(3)
	feeer := NewFixedFeeer(sa.Address, fee)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(t.cid, NewBig(99), NewTestAddress(), feeer)))

	opr := t.processor(cp, pool)

	sent := NewBig(31)

	items := []TransfersItem{t.newTransfersItem(ra.Address, sent)}
	tf := t.newTransfer(sa.Address, sa.Privs(), items)

	err := opr.Process(tf)

	var oper operation.ReasonError
	t.True(errors.As(err, &oper))
	t.Contains(err.Error(), "insufficient balance")
}

func (t *testTransfersOperations) TestSameSenders() {
	sa, st0 := t.newAccount(true, []Amount{NewAmount(NewBig(3), t.cid)})
	ra0, st1 := t.newAccount(true, []Amount{NewAmount(NewBig(1), t.cid)})
	ra1, st2 := t.newAccount(true, []Amount{NewAmount(NewBig(1), t.cid)})

	pool, _ := t.statepool(st0, st1, st2)
	feeer := NewFixedFeeer(sa.Address, ZeroBig)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(t.cid, NewBig(99), NewTestAddress(), feeer)))

	opr := t.processor(cp, pool)

	items := []TransfersItem{t.newTransfersItem(ra0.Address, NewBig(1))}
	tf0 := t.newTransfer(sa.Address, sa.Privs(), items)

	t.NoError(opr.Process(tf0))

	items = []TransfersItem{t.newTransfersItem(ra1.Address, NewBig(1))}
	tf1 := t.newTransfer(sa.Address, sa.Privs(), items)
	err := opr.Process(tf1)
	t.Contains(err.Error(), "violates only one sender")
}

func (t *testTransfersOperations) TestUnderThreshold() {
	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey := t.newKey(spk.Publickey(), 50)
	rkey := t.newKey(rpk.Publickey(), 50)
	skeys, _ := NewKeys([]Key{skey, rkey}, 100)
	rkeys, _ := NewKeys([]Key{rkey}, 50)

	pks := []key.Privatekey{spk}
	sender, _ := NewAddressFromKeys(skeys)
	receiver, _ := NewAddressFromKeys(rkeys)

	// set sender state
	senderBalance := NewAmount(NewBig(33), t.cid)

	var sts []state.State
	sts = append(sts,
		t.newStateBalance(sender, senderBalance.Big(), senderBalance.Currency()),
		t.newStateKeys(sender, skeys),
		t.newStateBalance(receiver, NewBig(3), t.cid),
		t.newStateKeys(receiver, skeys),
	)

	pool, _ := t.statepool(sts)
	feeer := NewFixedFeeer(sender, ZeroBig)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(t.cid, NewBig(99), NewTestAddress(), feeer)))

	opr := t.processor(cp, pool)

	items := []TransfersItem{t.newTransfersItem(receiver, NewBig(1))}
	tf := t.newTransfer(sender, pks, items)

	err := opr.Process(tf)

	var oper operation.ReasonError
	t.True(errors.As(err, &oper))
	t.Contains(err.Error(), "not passed threshold")
}

func (t *testTransfersOperations) TestUnknownKey() {
	sa, st0 := t.newAccount(true, []Amount{NewAmount(NewBig(1), t.cid)})
	ra, st1 := t.newAccount(true, []Amount{NewAmount(NewBig(1), t.cid)})

	pool, _ := t.statepool(st0, st1)
	feeer := NewFixedFeeer(sa.Address, ZeroBig)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(t.cid, NewBig(99), NewTestAddress(), feeer)))

	opr := t.processor(cp, pool)

	items := []TransfersItem{t.newTransfersItem(ra.Address, NewBig(1))}

	tf := t.newTransfer(sa.Address, []key.Privatekey{sa.Priv, key.MustNewBTCPrivatekey()}, items)

	err := opr.Process(tf)

	var oper operation.ReasonError
	t.True(errors.As(err, &oper))
	t.Contains(err.Error(), "unknown key found")
}

type acerr struct {
	err error
	ac  interface{}
	st  []state.State
}

func (ac acerr) Error() string {
	return ac.err.Error()
}

func (t *testTransfersOperations) TestConcurrentOperationsProcessor() {
	size := 100
	t.Equal(0, size%2)

	t.T().Log("size:", size)

	var started time.Time

	t.T().Log("trying to make accounts")
	started = time.Now()

	errchan := make(chan error)
	wk := util.NewDistributeWorker(100, errchan)

	go func() {
		wk.Run(
			func(i uint, j interface{}) error {
				if j == nil {
					return nil
				}
				ac, st := t.newAccount(true, []Amount{NewAmount(NewBig(int64(size)), t.cid)})

				return acerr{err: nil, ac: ac, st: st}
			},
		)

		close(errchan)
	}()

	go func() {
		for i := 0; i < size; i++ {
			wk.NewJob(i)
		}
		wk.Done(true)
	}()

	acs := make([]*account, size)
	sts := make([][]state.State, size)
	var i int
	for err := range errchan {
		if err == nil {
			continue
		}

		acs[i] = err.(acerr).ac.(*account)
		sts[i] = err.(acerr).st
		i++
	}
	t.T().Log("accounts created: ", len(acs), "elapsed:", time.Since(started))

	t.T().Log("trying to create operations")
	started = time.Now()

	errchan = make(chan error)
	wk = util.NewDistributeWorker(500, errchan)

	var items, ignored int64

	fee := NewBig(1)
	feeer := NewFixedFeeer(acs[0].Address, fee)

	half := size / 2
	go func() {
		wk.Run(
			func(_ uint, j interface{}) error {
				if j == nil {
					return nil
				}

				i := j.(int)
				var receiver *account
				if i == 0 || i == half {
					return acerr{err: nil}
				} else if i < half {
					receiver = acs[0]
				} else {
					receiver = acs[half]
				}

				//if i%2 != 0 {
				//	return acerr{err: nil}
				//}

				atomic.AddInt64(&items, 1)

				var ops []operation.Operation
				tf := t.newTransfer(acs[i].Address, acs[i].Privs(), []TransfersItem{t.newTransfersItem(receiver.Address, NewBig(1))})
				ops = append(ops, tf)

				if i%3 == 0 {
					tf := t.newTransfer(acs[i].Address, acs[i].Privs(), []TransfersItem{t.newTransfersItem(receiver.Address, NewBig(int64(size+1)))})
					ops = append(ops, tf)

					atomic.AddInt64(&ignored, 1)
				}

				return acerr{err: nil, ac: ops}
			},
		)

		close(errchan)
	}()

	go func() {
		for i := 0; i < size; i++ {
			wk.NewJob(i)
		}
		wk.Done(true)
	}()

	var ops []operation.Operation
	for err := range errchan {
		if err == nil || err.(acerr).ac == nil {
			continue
		}

		op := err.(acerr).ac.([]operation.Operation)
		ops = append(ops, op...)
	}
	t.T().Log("operations created:", len(ops), "elapsed:", time.Since(started))

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(t.cid, NewBig(99), NewTestAddress(), feeer)))

	copr, err := NewOperationProcessor(cp).
		SetProcessor(Transfers{}, NewTransfersProcessor(cp))
	t.NoError(err)

	oppHintSet := hint.NewHintmap()
	t.NoError(oppHintSet.Add(Transfers{}, copr))

	pool, _ := t.statepool(sts...)

	t.T().Log("trying to process")
	started = time.Now()

	co, err := prprocessor.NewConcurrentOperationsProcessor(uint64(len(ops)), 100, pool, oppHintSet)
	t.NoError(err)
	co.Start(context.Background(), nil)

	for i := range ops {
		t.NoError(co.Process(uint64(i), ops[i]))
	}
	t.NoError(co.Close())

	// co := &defaultOperationProcessor{pool: t.pool}
	//
	// Or,
	//
	// co := &OperationProcessor{pool: t.pool}
	// for _, op := range ops {
	// 	t.NoError(co.Process(op))
	// }

	t.T().Log("procsseed, elapsed:", time.Since(started))

	result := map[string]state.State{}
	for _, st := range pool.Updates() {
		result[st.Key()] = st.GetState()
	}

	for i, ac := range acs {
		b, found := result[StateKeyBalance(ac.Address, t.cid)]
		t.True(found)
		a, err := StateBalanceValue(b)
		t.NoError(err)

		var expected Big
		if i == 0 || i == half {
			expected = NewBig(int64(size)).Add(NewBig(int64(half) - 1))
			if i == 0 {
				expected = expected.Add(NewBig(atomic.LoadInt64(&items)))
			}

		} else {
			expected = NewBig(int64(size)).Sub(NewBig(1)).Sub(fee)
		}

		t.Equal(expected, a.Big(), i)
	}

	tr, err := co.OperationsTree()
	t.NoError(err)

	var notInState int64
	t.NoError(tr.Traverse(func(no tree.FixedTreeNode) (bool, error) {
		ono, ok := no.(operation.FixedTreeNode)
		t.True(ok)

		if !ono.InState() {
			notInState++
		}

		return true, nil
	}))
	t.Equal(atomic.LoadInt64(&ignored), notInState)
}

// TODO write benchmark for OperationProcessor

func TestTransfersOperations(t *testing.T) {
	suite.Run(t, new(testTransfersOperations))
}
