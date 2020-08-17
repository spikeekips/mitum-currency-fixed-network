package currency

import (
	"fmt"
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
	"github.com/spikeekips/mitum/util/valuehash"
)

type account struct {
	Address base.Address
	Priv    key.Privatekey
	Key     Key
}

func (ac *account) Transfer(receiver base.Address, amount Amount) Transfer {
	token := util.UUID().Bytes()
	fact := NewTransferFact(token, ac.Address, receiver, amount)

	var fs []operation.FactSign
	if sig, err := operation.NewFactSignature(ac.Priv, fact, nil); err != nil {
		panic(err)
	} else {
		fs = []operation.FactSign{operation.NewBaseFactSign(ac.Priv.Publickey(), sig)}
	}

	if tf, err := NewTransfer(fact, fs, ""); err != nil {
		panic(err)
	} else {
		return tf
	}
}

func generateAccount() *account {
	priv := key.MustNewBTCPrivatekey()

	key := NewKey(priv.Publickey(), 100)
	address, _ := NewAddressFromKeys([]Key{key})

	return &account{Address: address, Priv: priv, Key: key}
}

type baseTestOperationProcessor struct {
	suite.Suite
	process func(*isaac.Statepool, state.StateProcessor) error
}

func (t *baseTestOperationProcessor) SetupSuite() {
	if t.process == nil {
		t.process = func(sp *isaac.Statepool, po state.StateProcessor) error {
			return po.Process(sp.Get, sp.Set)
		}
	}
}

type testTransferOperation struct {
	suite.Suite
	isaac.StorageSupportTest
	baseTestOperationProcessor
	co *ConcurrentOperationsProcessor
}

func (t *testTransferOperation) SetupSuite() {
	t.StorageSupportTest.SetupSuite()

	t.Encs.AddHinter(key.BTCPublickey{})
	t.Encs.AddHinter(operation.BaseFactSign{})
	t.Encs.AddHinter(Key{})
	t.Encs.AddHinter(Keys{})
	t.Encs.AddHinter(Address(""))
	t.Encs.AddHinter(Transfer{})

	t.baseTestOperationProcessor.SetupSuite()
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

func (t *testTransferOperation) newStateAccount(a base.Address, amount Amount, sp *isaac.Statepool) state.StateUpdater {
	key := StateKeyBalance(a)
	value, _ := state.NewStringValue(amount.String())
	su, err := state.NewStateV0Updater(key, value, valuehash.RandomSHA256())
	t.NoError(err)

	t.NoError(sp.Set(valuehash.RandomSHA256(), su))

	ust, found, err := sp.Get(key)
	t.NoError(err)
	t.NotNil(ust)
	t.True(found)

	return su
}

func (t *testTransferOperation) newStateKeys(a base.Address, keys Keys, sp *isaac.Statepool) state.StateUpdater {
	key := StateKeyKeys(a)
	value, _ := state.NewHintedValue(keys)
	su, err := state.NewStateV0Updater(key, value, valuehash.RandomSHA256())
	t.NoError(err)

	t.NoError(sp.Set(valuehash.RandomSHA256(), su))

	ust, found, err := sp.Get(key)
	t.NoError(err)
	t.NotNil(ust)
	t.True(found)

	return su
}

func (t *testTransferOperation) TestSenderNotExist() {
	sp, err := isaac.NewStatepool(t.Storage(nil, nil))
	t.NoError(err)

	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey := NewKey(spk.Publickey(), 100)
	rkey := NewKey(rpk.Publickey(), 100)

	pks := []key.Privatekey{spk, rpk}
	sender, _ := NewAddressFromKeys([]Key{skey})
	receiver, _ := NewAddressFromKeys([]Key{rkey})

	tf := t.newTransfer(sender, receiver, NewAmount(10), pks)

	err = t.process(sp, tf)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "keys of sender does not exist")
}

func (t *testTransferOperation) TestReceiverNotExist() {
	sp, err := isaac.NewStatepool(t.Storage(nil, nil))
	t.NoError(err)

	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey := NewKey(spk.Publickey(), 100)
	rkey := NewKey(rpk.Publickey(), 100)
	skeys, _ := NewKeys([]Key{skey, rkey}, 100)

	pks := []key.Privatekey{spk, rpk}
	sender, _ := NewAddressFromKeys([]Key{skey})
	receiver, _ := NewAddressFromKeys([]Key{rkey})

	// set sender state
	_ = t.newStateAccount(sender, NewAmount(10), sp)
	_ = t.newStateKeys(sender, skeys, sp)

	tf := t.newTransfer(sender, receiver, NewAmount(3), pks)

	err = t.process(sp, tf)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "keys of receiver does not exist")
}

func (t *testTransferOperation) TestInsufficientBalance() {
	sp, err := isaac.NewStatepool(t.Storage(nil, nil))
	t.NoError(err)

	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey := NewKey(spk.Publickey(), 50)
	rkey := NewKey(rpk.Publickey(), 50)
	skeys, _ := NewKeys([]Key{skey, rkey}, 100)

	pks := []key.Privatekey{spk, rpk}
	sender, _ := NewAddressFromKeys([]Key{skey})
	receiver, _ := NewAddressFromKeys([]Key{rkey})

	// set sender state
	senderBalance := NewAmount(10)
	amount := NewAmount(33)

	_ = t.newStateAccount(sender, senderBalance, sp)
	_ = t.newStateKeys(sender, skeys, sp)
	_ = t.newStateAccount(receiver, NewAmount(3), sp)
	_ = t.newStateKeys(receiver, skeys, sp)

	tf := t.newTransfer(sender, receiver, amount, pks)

	err = t.process(sp, tf)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "invalid amount; under zero")
}

func (t *testTransferOperation) TestSufficientBalance() {
	sp, err := isaac.NewStatepool(t.Storage(nil, nil))
	t.NoError(err)

	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey := NewKey(spk.Publickey(), 50)
	rkey := NewKey(rpk.Publickey(), 50)
	rkeys, _ := NewKeys([]Key{skey, rkey}, 100)
	skeys, _ := NewKeys([]Key{skey, rkey}, 100)

	pks := []key.Privatekey{spk, rpk}
	sender, _ := NewAddressFromKeys([]Key{skey})
	receiver, _ := NewAddressFromKeys([]Key{rkey})

	// set sender state
	senderBalance := NewAmount(33)
	amount := NewAmount(10)

	_ = t.newStateAccount(sender, senderBalance, sp)
	_ = t.newStateKeys(sender, skeys, sp)
	_ = t.newStateAccount(receiver, NewAmount(3), sp)
	_ = t.newStateKeys(receiver, rkeys, sp)

	tf := t.newTransfer(sender, receiver, amount, pks)

	err = t.process(sp, tf)
	t.NoError(err)

	// checking value
	sstate, found, err := sp.Get(StateKeyBalance(sender))
	t.NoError(err)
	t.True(found)
	t.NotNil(sstate)

	rstate, found, err := sp.Get(StateKeyBalance(receiver))
	t.NoError(err)
	t.True(found)
	t.NotNil(rstate)

	t.Equal(sstate.Value().Interface(), "23")
	t.Equal(rstate.Value().Interface(), "13")
}

func (t *testTransferOperation) TestUnderThreshold() {
	sp, err := isaac.NewStatepool(t.Storage(nil, nil))
	t.NoError(err)

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

	_ = t.newStateAccount(sender, senderBalance, sp)
	_ = t.newStateKeys(sender, skeys, sp)
	_ = t.newStateAccount(receiver, NewAmount(3), sp)
	_ = t.newStateKeys(receiver, skeys, sp)

	tf := t.newTransfer(sender, receiver, amount, pks)

	err = t.process(sp, tf)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "not passed threshold")
}

func (t *testTransferOperation) TestUnknownKey() {
	sp, err := isaac.NewStatepool(t.Storage(nil, nil))
	t.NoError(err)

	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey := NewKey(spk.Publickey(), 50)
	rkey := NewKey(rpk.Publickey(), 50)
	skeys, _ := NewKeys([]Key{skey, rkey}, 100)

	sender, _ := NewAddressFromKeys([]Key{skey})
	receiver, _ := NewAddressFromKeys([]Key{rkey})

	// set sender state
	senderBalance := NewAmount(33)
	amount := NewAmount(10)

	_ = t.newStateAccount(sender, senderBalance, sp)
	_ = t.newStateKeys(sender, skeys, sp)
	_ = t.newStateAccount(receiver, NewAmount(3), sp)
	_ = t.newStateKeys(receiver, skeys, sp)

	tf := t.newTransfer(sender, receiver, amount, []key.Privatekey{spk, key.MustNewBTCPrivatekey()})

	err = t.process(sp, tf)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "unknown key found")
}

func TestTransferOperation(t *testing.T) {
	suite.Run(t, new(testTransferOperation))
}

func TestTransferOperationProcessor(t *testing.T) {
	n := new(testTransferOperation)

	_ = (interface{})(&OperationProcessor{}).(isaac.OperationProcessor)

	n.process = func(sp *isaac.Statepool, op state.StateProcessor) error {
		opr := (&OperationProcessor{})
		return opr.New(sp).Process(op)
	}
	suite.Run(t, n)
}

type acerr struct {
	err error
	ac  interface{}
}

func (ac acerr) Error() string {
	return ac.err.Error()
}

func (t *testTransferOperation) TestRemoveMe() {
	sp, err := isaac.NewStatepool(t.Storage(nil, nil))
	t.NoError(err)

	size := 30000

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
				_ = t.newStateKeys(ac.Address, keys, sp)
				_ = t.newStateAccount(ac.Address, NewAmount(int64(size)), sp)

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

				return acerr{err: nil, ac: acs[0].Transfer(acs[i].Address, NewAmount(1))}
			},
		)

		close(errchan)
	}()

	go func() {
		for i := 1; i < size; i++ {
			wk.NewJob(i)
		}
		wk.Done(true)
	}()

	ops := make([]state.StateProcessor, size-1)
	i = 0
	for err := range errchan {
		if err == nil {
			continue
		}

		op := err.(acerr).ac.(state.StateProcessor)
		ops[i] = op
		i++
	}

	oppHintSet := hint.NewHintmap()
	t.NoError(oppHintSet.Add(Transfer{}, &OperationProcessor{}))

	baseSt := map[string]state.State{}
	for _, st := range sp.Updates() {
		baseSt[st.Key()] = st
	}

	sp, err = isaac.NewStatepoolWithBase(t.Storage(nil, nil), baseSt)
	t.NoError(err)

	started := time.Now()
	co, err := NewConcurrentOperationsProcessor(100, sp, time.Second*10, oppHintSet)
	t.NoError(err)
	co.start()

	for _, op := range ops {
		t.NoError(co.Process(op))
	}
	t.NoError(co.Close())

	// co := &defaultOperationProcessor{pool: sp}

	//co := &OperationProcessor{pool: sp}
	//for _, op := range ops {
	//	t.NoError(co.Process(op))
	//}

	fmt.Println("elapsed:", time.Since(started))

	for i, ac := range acs {
		b, err := existsAccountState(StateKeyBalance(ac.Address), "", sp.Get)
		t.NoError(err)
		a, err := StateAmountValue(b)
		t.NoError(err)

		if i == 0 {
			t.Equal(NewAmount(1), a, i)
		} else {
			t.Equal(NewAmount(int64(size)+1), a, i)
		}
	}
}
