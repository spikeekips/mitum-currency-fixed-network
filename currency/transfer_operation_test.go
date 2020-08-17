package currency

import (
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

type testTransferOperation struct {
	baseTestOperationProcessor
}

func (t *testTransferOperation) newTransfer(sender, receiver base.Address, amount Amount, keys []key.Privatekey) Transfer {
	token := util.UUID().Bytes()
	fact := NewTransferFact(token, sender, receiver, amount)

	var fs []operation.FactSign
	for _, pk := range keys {
		sig, err := operation.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
	}

	tf, err := NewTransfer(fact, fs, "")
	t.NoError(err)

	t.NoError(tf.IsValid(nil))

	return tf
}

func (t *testTransferOperation) TestSenderNotExist() {
	sa := t.newAccount(false, NilAmount)
	ra := t.newAccount(false, NilAmount)

	tf := t.newTransfer(sa.Address, ra.Address, NewAmount(10), sa.Privs())

	err := t.opr.Process(tf)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "does not exist")
}

func (t *testTransferOperation) TestReceiverNotExist() {
	sa := t.newAccount(true, NewAmount(10))
	ra := t.newAccount(false, NilAmount)

	tf := t.newTransfer(sa.Address, ra.Address, NewAmount(3), sa.Privs())

	err := t.opr.Process(tf)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "keys of receiver does not exist")
}

func (t *testTransferOperation) TestInsufficientBalance() {
	saBalance := NewAmount(10)
	sa := t.newAccount(true, saBalance)
	ra := t.newAccount(true, NewAmount(1))

	tf := t.newTransfer(sa.Address, ra.Address, saBalance.Add(NewAmount(1)), sa.Privs())

	err := t.opr.Process(tf)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "insufficient balance")
}

func (t *testTransferOperation) TestSufficientBalance() {
	saBalance := NewAmount(33)
	raBalance := NewAmount(1)
	sa := t.newAccount(true, saBalance)
	ra := t.newAccount(true, NewAmount(1))

	sent := saBalance.Sub(NewAmount(10))

	tf := t.newTransfer(sa.Address, ra.Address, sent, sa.Privs())

	err := t.opr.Process(tf)
	t.NoError(err)

	// checking value
	sstate, found, err := t.pool.Get(StateKeyBalance(sa.Address))
	t.NoError(err)
	t.True(found)
	t.NotNil(sstate)

	rstate, found, err := t.pool.Get(StateKeyBalance(ra.Address))
	t.NoError(err)
	t.True(found)
	t.NotNil(rstate)

	t.Equal(sstate.Value().Interface(), saBalance.Sub(sent).String())
	t.Equal(rstate.Value().Interface(), raBalance.Add(sent).String())
}

func (t *testTransferOperation) TestSameSenders() {
	sa := t.newAccount(true, NewAmount(3))
	ra := t.newAccount(true, NewAmount(1))

	tf0 := t.newTransfer(sa.Address, ra.Address, NewAmount(1), sa.Privs())

	t.NoError(t.opr.Process(tf0))

	tf1 := t.newTransfer(sa.Address, ra.Address, NewAmount(1), sa.Privs())
	err := t.opr.Process(tf1)
	t.Contains(err.Error(), "violates only one sender")
}

func (t *testTransferOperation) TestUnderThreshold() {
	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey := NewKey(spk.Publickey(), 50)
	rkey := NewKey(rpk.Publickey(), 50)
	skeys, _ := NewKeys([]Key{skey, rkey}, 100)

	pks := []key.Privatekey{spk}
	sender, _ := NewAddressFromKeys([]Key{skey})
	receiver, _ := NewAddressFromKeys([]Key{rkey})

	// set sender state
	senderBalance := NewAmount(33)
	amount := NewAmount(10)

	_ = t.newStateBalance(sender, senderBalance)
	_ = t.newStateKeys(sender, skeys)
	_ = t.newStateBalance(receiver, NewAmount(3))
	_ = t.newStateKeys(receiver, skeys)

	tf := t.newTransfer(sender, receiver, amount, pks)

	err := t.opr.Process(tf)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "not passed threshold")
}

func (t *testTransferOperation) TestUnknownKey() {
	sa := t.newAccount(true, NewAmount(1))
	ra := t.newAccount(true, NewAmount(1))

	tf := t.newTransfer(sa.Address, ra.Address, NewAmount(1), []key.Privatekey{sa.Priv, key.MustNewBTCPrivatekey()})

	err := t.opr.Process(tf)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "unknown key found")
}

type acerr struct {
	err error
	ac  interface{}
}

func (ac acerr) Error() string {
	return ac.err.Error()
}

func (t *testTransferOperation) TestConcurrentOperationsProcessor() {
	t.T().Skip()

	size := 30000
	t.Equal(0, size%2)

	errchan := make(chan error)
	wk := util.NewDistributeWorker(100, errchan)

	go func() {
		wk.Run(
			func(i uint, j interface{}) error {
				if j == nil {
					return nil
				}
				ac := generateAccount()
				keys, _ := NewKeys([]Key{ac.Key}, 100)
				_ = t.newStateKeys(ac.Address, keys)
				_ = t.newStateBalance(ac.Address, NewAmount(int64(size)))

				return acerr{err: nil, ac: ac}
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
	var i int
	for err := range errchan {
		if err == nil {
			continue
		}

		acs[i] = err.(acerr).ac.(*account)
		i++
	}

	errchan = make(chan error)
	wk = util.NewDistributeWorker(100, errchan)

	go func() {
		wk.Run(
			func(_ uint, j interface{}) error {
				if j == nil {
					return nil
				}

				i := j.(int)
				var receiver *account
				if len(acs) == i+1 {
					receiver = acs[0]
				} else {
					receiver = acs[i+1]
				}

				if i%2 != 0 {
					return acerr{err: nil}
				}

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

	var ops []state.StateProcessor
	for err := range errchan {
		if err == nil || err.(acerr).ac == nil {
			continue
		}

		op := err.(acerr).ac.(state.StateProcessor)
		ops = append(ops, op)
	}

	oppHintSet := hint.NewHintmap()
	t.NoError(oppHintSet.Add(Transfer{}, &OperationProcessor{}))

	baseSt := map[string]state.State{}
	for _, st := range t.pool.Updates() {
		baseSt[st.Key()] = st
	}

	pool, err := isaac.NewStatepoolWithBase(t.Storage(nil, nil), baseSt)
	t.NoError(err)
	t.pool = pool

	started := time.Now()
	co, err := NewConcurrentOperationsProcessor(100, t.pool, time.Second*10, oppHintSet)
	t.NoError(err)
	co.Start()

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

	t.T().Log("elapsed:", time.Since(started))

	for i, ac := range acs {
		b, err := existsAccountState(StateKeyBalance(ac.Address), "", t.pool.Get)
		t.NoError(err)
		a, err := StateAmountValue(b)
		t.NoError(err)

		var expected Amount
		if i%2 == 0 {
			expected = NewAmount(int64(size)).Sub(NewAmount(1))
		} else {
			expected = NewAmount(int64(size)).Add(NewAmount(1))
		}

		t.Equal(expected, a, i)
	}
}

// TODO write benchmark for OperationProcessor

func TestTransferOperation(t *testing.T) {
	suite.Run(t, new(testTransferOperation))
}
