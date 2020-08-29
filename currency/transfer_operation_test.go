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
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/isaac"
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

	_, opr := t.statepool()

	err := opr.Process(tf)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "does not exist")
}

func (t *testTransfersOperations) TestReceiverNotExist() {
	sa, sts := t.newAccount(true, NewAmount(10))
	ra, _ := t.newAccount(false, NilAmount)

	_, opr := t.statepool(sts)

	tf := t.newTransfer(sa.Address, ra.Address, NewAmount(3), sa.Privs())

	err := opr.Process(tf)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "keys of receiver does not exist")
}

func (t *testTransfersOperations) TestInsufficientBalance() {
	saBalance := NewAmount(10)
	sa, st0 := t.newAccount(true, saBalance)
	ra, st1 := t.newAccount(true, NewAmount(1))

	_, opr := t.statepool(st0, st1)

	tf := t.newTransfer(sa.Address, ra.Address, saBalance.Add(NewAmount(1)), sa.Privs())

	err := opr.Process(tf)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "insufficient balance")
}

func (t *testTransfersOperations) TestSufficientBalance() {
	saBalance := NewAmount(33)
	raBalance := NewAmount(1)
	sa, st0 := t.newAccount(true, saBalance)
	ra, st1 := t.newAccount(true, NewAmount(1))

	pool, opr := t.statepool(st0, st1)

	sent := saBalance.Sub(NewAmount(10))

	tf := t.newTransfer(sa.Address, ra.Address, sent, sa.Privs())

	err := opr.Process(tf)
	t.NoError(err)

	var sst, rst state.State
	for _, st := range pool.Updates() {
		if st.Key() == StateKeyBalance(sa.Address) {
			sst = st
		} else if st.Key() == StateKeyBalance(ra.Address) {
			rst = st
		}
	}

	// checking value
	t.Equal(sst.Value().Interface(), saBalance.Sub(sent).String())
	t.Equal(rst.Value().Interface(), raBalance.Add(sent).String())
}

func (t *testTransfersOperations) TestSameSenders() {
	sa, st0 := t.newAccount(true, NewAmount(3))
	ra, st1 := t.newAccount(true, NewAmount(1))

	_, opr := t.statepool(st0, st1)

	tf0 := t.newTransfer(sa.Address, ra.Address, NewAmount(1), sa.Privs())

	t.NoError(opr.Process(tf0))

	tf1 := t.newTransfer(sa.Address, ra.Address, NewAmount(1), sa.Privs())
	err := opr.Process(tf1)
	t.Contains(err.Error(), "violates only one sender")
}

func (t *testTransfersOperations) TestUnderThreshold() {
	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey := NewKey(spk.Publickey(), 50)
	rkey := NewKey(rpk.Publickey(), 50)
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

	_, opr := t.statepool(sts)

	tf := t.newTransfer(sender, receiver, amount, pks)

	err := opr.Process(tf)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "not passed threshold")
}

func (t *testTransfersOperations) TestUnknownKey() {
	sa, st0 := t.newAccount(true, NewAmount(1))
	ra, st1 := t.newAccount(true, NewAmount(1))

	_, opr := t.statepool(st0, st1)

	tf := t.newTransfer(sa.Address, ra.Address, NewAmount(1), []key.Privatekey{sa.Priv, key.MustNewBTCPrivatekey()})

	err := opr.Process(tf)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
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

	oppHintSet := hint.NewHintmap()
	t.NoError(oppHintSet.Add(Transfers{}, &OperationProcessor{}))

	pool, _ := t.statepool(sts...)

	t.T().Log("trying to process")
	started = time.Now()

	co, err := isaac.NewConcurrentOperationsProcessor(100, pool, oppHintSet)
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
		result[st.Key()] = st
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
			expected = NewAmount(int64(size)).Sub(NewAmount(1))
		}

		t.Equal(expected, a, i)
	}
}

// TODO write benchmark for OperationProcessor

func TestTransfersOperations(t *testing.T) {
	suite.Run(t, new(testTransfersOperations))
}
