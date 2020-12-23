package currency

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
)

type testTransfersOperations struct {
	baseTestOperationProcessor
}

func (t *testTransfersOperations) newTransfer(sender, receiver base.Address, amount Amount, keys []key.Privatekey) Transfers {
	token := util.UUID().Bytes()
	items := []TransferItem{NewTransferItem(receiver, amount)}
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
	sa, _ := t.newAccount(false, NilAmount)
	ra, _ := t.newAccount(false, NilAmount)

	tf := t.newTransfer(sa.Address, ra.Address, NewAmount(10), sa.Privs())

	pool, _ := t.statepool()
	opr := NewOperationProcessor(NewFixedFeeAmount(ZeroAmount), func() (base.Address, error) { return sa.Address, nil }).New(pool)

	err := opr.Process(tf)
	t.True(xerrors.Is(err, util.IgnoreError))
	t.Contains(err.Error(), "does not exist")
}

func (t *testTransfersOperations) TestReceiverNotExist() {
	sa, sts := t.newAccount(true, NewAmount(10))
	ra, _ := t.newAccount(false, NilAmount)

	pool, _ := t.statepool(sts)
	opr := NewOperationProcessor(NewFixedFeeAmount(ZeroAmount), func() (base.Address, error) { return sa.Address, nil }).New(pool)

	tf := t.newTransfer(sa.Address, ra.Address, NewAmount(3), sa.Privs())

	err := opr.Process(tf)
	t.True(xerrors.Is(err, util.IgnoreError))
	t.Contains(err.Error(), "keys of receiver does not exist")
}

func (t *testTransfersOperations) TestInsufficientBalance() {
	saBalance := NewAmount(10)
	sa, st0 := t.newAccount(true, saBalance)
	ra, st1 := t.newAccount(true, NewAmount(1))

	pool, _ := t.statepool(st0, st1)
	opr := NewOperationProcessor(NewFixedFeeAmount(ZeroAmount), func() (base.Address, error) { return sa.Address, nil }).New(pool)

	tf := t.newTransfer(sa.Address, ra.Address, saBalance.Add(NewAmount(1)), sa.Privs())

	err := opr.Process(tf)
	t.True(xerrors.Is(err, util.IgnoreError))
	t.Contains(err.Error(), "insufficient balance")
}

func (t *testTransfersOperations) TestSufficientBalance() {
	faBalance := NewAmount(22)
	saBalance := NewAmount(33)
	raBalance := NewAmount(1)
	fa, st0 := t.newAccount(true, faBalance)
	sa, st1 := t.newAccount(true, saBalance)
	ra, st2 := t.newAccount(true, NewAmount(1))

	pool, _ := t.statepool(st0, st1, st2)

	fee := NewAmount(2)
	opr := NewOperationProcessor(NewFixedFeeAmount(fee), func() (base.Address, error) { return fa.Address, nil }).New(pool)

	sent := saBalance.Sub(NewAmount(10))

	tf := t.newTransfer(sa.Address, ra.Address, sent, sa.Privs())

	t.NoError(opr.Process(tf))
	t.NoError(opr.Close())

	var sst, rst, fst state.State
	for _, st := range pool.Updates() {
		if st.Key() == StateKeyBalance(sa.Address) {
			sst = st.GetState()
		} else if st.Key() == StateKeyBalance(ra.Address) {
			rst = st.GetState()
		} else if st.Key() == StateKeyBalance(fa.Address) {
			fst = st.GetState()
		}
	}

	// checking value
	t.Equal(sst.Value().Interface(), saBalance.Sub(sent).Sub(fee).String())
	t.Equal(rst.Value().Interface(), raBalance.Add(sent).String())
	t.Equal(fst.Value().Interface(), faBalance.Add(fee).String())

	// check fee operation

	t.True(len(pool.AddedOperations()) > 0)
	var fo FeeOperation
	for _, op := range pool.AddedOperations() {
		if err := op.Hint().IsCompatible(FeeOperationHint); err == nil {
			fo = op.(FeeOperation)
		}
	}

	fof := fo.Fact().(FeeOperationFact)
	t.Equal(fee, fof.Fee())
}

func (t *testTransfersOperations) TestMultipleItemsWithFee() {
	saBalance := NewAmount(33)
	sa, st0 := t.newAccount(true, saBalance)
	ra0, rst0 := t.newAccount(true, NewAmount(0))
	ra1, rst1 := t.newAccount(true, NewAmount(0))

	pool, _ := t.statepool(st0, rst0, rst1)

	fee := NewAmount(2)
	opr := NewOperationProcessor(NewFixedFeeAmount(fee), func() (base.Address, error) { return sa.Address, nil }).New(pool)

	sent := NewAmount(10)

	token := util.UUID().Bytes()
	items := []TransferItem{
		NewTransferItem(ra0.Address, sent),
		NewTransferItem(ra1.Address, sent),
	}
	fact := NewTransfersFact(token, sa.Address, items)
	sig, err := operation.NewFactSignature(sa.Privs()[0], fact, nil)
	t.NoError(err)
	fs := []operation.FactSign{operation.NewBaseFactSign(sa.Privs()[0].Publickey(), sig)}
	tf, err := NewTransfers(fact, fs, "")
	t.NoError(err)

	err = opr.Process(tf)
	t.NoError(err)

	var nst, nrst0, nrst1 state.State
	for _, st := range pool.Updates() {
		if st.Key() == StateKeyBalance(sa.Address) {
			nst = st.GetState()
		} else if st.Key() == StateKeyBalance(ra0.Address) {
			nrst0 = st.GetState()
		} else if st.Key() == StateKeyBalance(ra1.Address) {
			nrst1 = st.GetState()
		}
	}

	t.Equal(saBalance.Sub(sent.MulInt64(2)).Sub(fee.MulInt64(2)).String(), nst.Value().Interface())
	t.Equal(sent.String(), nrst0.Value().Interface())
	t.Equal(sent.String(), nrst1.Value().Interface())
	t.Equal(fee.MulInt64(2), nst.(AmountState).Fee())
}

func (t *testTransfersOperations) TestInsufficientMultipleItemsWithFee() {
	saBalance := NewAmount(33)
	sa, st0 := t.newAccount(true, saBalance)
	ra0, rst0 := t.newAccount(true, NewAmount(0))
	ra1, rst1 := t.newAccount(true, NewAmount(0))

	pool, _ := t.statepool(st0, rst0, rst1)

	fee := NewAmount(2)
	opr := NewOperationProcessor(NewFixedFeeAmount(fee), func() (base.Address, error) { return sa.Address, nil }).New(pool)

	sent := NewAmount(15)

	token := util.UUID().Bytes()
	items := []TransferItem{
		NewTransferItem(ra0.Address, sent),
		NewTransferItem(ra1.Address, sent),
	}
	fact := NewTransfersFact(token, sa.Address, items)
	sig, err := operation.NewFactSignature(sa.Privs()[0], fact, nil)
	t.NoError(err)
	fs := []operation.FactSign{operation.NewBaseFactSign(sa.Privs()[0].Publickey(), sig)}
	tf, err := NewTransfers(fact, fs, "")
	t.NoError(err)

	err = opr.Process(tf)
	t.True(xerrors.Is(err, util.IgnoreError))
	t.Contains(err.Error(), "insufficient balance")
}

func (t *testTransfersOperations) TestInSufficientBalanceWithFee() {
	saBalance := NewAmount(33)
	sa, st0 := t.newAccount(true, saBalance)
	ra, st1 := t.newAccount(true, NewAmount(1))

	pool, _ := t.statepool(st0, st1)

	fee := NewAmount(3)
	opr := NewOperationProcessor(NewFixedFeeAmount(fee), func() (base.Address, error) { return sa.Address, nil }).New(pool)

	sent := NewAmount(31)
	tf := t.newTransfer(sa.Address, ra.Address, sent, sa.Privs())

	err := opr.Process(tf)
	t.True(xerrors.Is(err, util.IgnoreError))
	t.Contains(err.Error(), "insufficient balance")
}

func (t *testTransfersOperations) TestSameSenders() {
	sa, st0 := t.newAccount(true, NewAmount(3))
	ra, st1 := t.newAccount(true, NewAmount(1))

	pool, _ := t.statepool(st0, st1)
	opr := NewOperationProcessor(NewFixedFeeAmount(ZeroAmount), func() (base.Address, error) { return sa.Address, nil }).New(pool)

	tf0 := t.newTransfer(sa.Address, ra.Address, NewAmount(1), sa.Privs())

	t.NoError(opr.Process(tf0))

	tf1 := t.newTransfer(sa.Address, ra.Address, NewAmount(1), sa.Privs())
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
	senderBalance := NewAmount(33)
	amount := NewAmount(10)

	var sts []state.State
	sts = append(sts,
		t.newStateBalance(sender, senderBalance),
		t.newStateKeys(sender, skeys),
		t.newStateBalance(receiver, NewAmount(3)),
		t.newStateKeys(receiver, skeys),
	)

	pool, _ := t.statepool(sts)
	opr := NewOperationProcessor(NewFixedFeeAmount(ZeroAmount), func() (base.Address, error) { return sender, nil }).New(pool)

	tf := t.newTransfer(sender, receiver, amount, pks)

	err := opr.Process(tf)
	t.True(xerrors.Is(err, util.IgnoreError))
	t.Contains(err.Error(), "not passed threshold")
}

func (t *testTransfersOperations) TestUnknownKey() {
	sa, st0 := t.newAccount(true, NewAmount(1))
	ra, st1 := t.newAccount(true, NewAmount(1))

	pool, _ := t.statepool(st0, st1)
	opr := NewOperationProcessor(NewFixedFeeAmount(ZeroAmount), func() (base.Address, error) { return sa.Address, nil }).New(pool)

	tf := t.newTransfer(sa.Address, ra.Address, NewAmount(1), []key.Privatekey{sa.Priv, key.MustNewBTCPrivatekey()})

	err := opr.Process(tf)
	t.True(xerrors.Is(err, util.IgnoreError))
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
				ac, st := t.newAccount(true, NewAmount(int64(size)))

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

				tf := t.newTransfer(acs[i].Address, receiver.Address, NewAmount(1), acs[i].Privs())

				return acerr{err: nil, ac: tf}
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

		op := err.(acerr).ac.(operation.Operation)
		ops = append(ops, op)
	}
	t.T().Log("operations created:", len(ops), "elapsed:", time.Since(started))

	fee := NewAmount(1)
	nopr := NewOperationProcessor(
		NewFixedFeeAmount(fee),
		func() (base.Address, error) { return acs[0].Address, nil }, // NOTE 1st account will get all fee
	)

	oppHintSet := hint.NewHintmap()
	t.NoError(oppHintSet.Add(Transfers{}, nopr))

	pool, _ := t.statepool(sts...)

	t.T().Log("trying to process")
	started = time.Now()

	co, err := prprocessor.NewConcurrentOperationsProcessor(100, pool, oppHintSet)
	t.NoError(err)
	co.Start(context.Background(), nil)

	for _, op := range ops {
		t.NoError(co.Process(op))
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
		b, found := result[StateKeyBalance(ac.Address)]
		t.True(found)
		a, err := StateAmountValue(b)
		t.NoError(err)

		var expected Amount
		if i == 0 || i == half {
			expected = NewAmount(int64(size)).Add(NewAmount(int64(half) - 1))
		} else {
			expected = NewAmount(int64(size)).Sub(NewAmount(1)).Sub(fee)
		}

		t.Equal(expected, a, i)
	}
}

// TODO write benchmark for OperationProcessor

func TestTransfersOperations(t *testing.T) {
	suite.Run(t, new(testTransfersOperations))
}
