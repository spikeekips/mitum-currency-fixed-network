package mc

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testTransferOperation struct {
	suite.Suite
	isaac.StorageSupportTest
}

func (t *testTransferOperation) SetupSuite() {
	t.StorageSupportTest.SetupSuite()

	t.Encs.AddHinter(key.BTCPublickey{})
	t.Encs.AddHinter(operation.BaseFactSign{})
	t.Encs.AddHinter(Key{})
	t.Encs.AddHinter(Keys{})
	t.Encs.AddHinter(Address(""))
	t.Encs.AddHinter(Transfer{})
}

func (t *testTransferOperation) newTransfer(sender, receiver Address, amount Amount, keys []key.Privatekey) Transfer {
	token := util.UUID().Bytes()
	fact := NewTransferFact(token, sender, receiver, amount)

	var fs []operation.FactSign
	for _, pk := range keys {
		sig, err := operation.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
	}

	tf, err := NewTransfer(fact, fs)
	t.NoError(err)

	t.NoError(tf.IsValid(nil))

	return tf
}

func (t *testTransferOperation) newStateAccount(a Address, amount Amount, sp *isaac.StatePool) state.StateUpdater {
	key := StateKeyBalance(a)
	value, _ := state.NewStringValue(amount.String())
	su, err := state.NewStateV0(key, value, valuehash.RandomSHA256())
	t.NoError(err)

	t.NoError(sp.Set(su))

	ust, found, err := sp.Get(key)
	t.NoError(err)
	t.NotNil(ust)
	t.True(found)

	return su
}

func (t *testTransferOperation) newStateKeys(a Address, keys Keys, sp *isaac.StatePool) state.StateUpdater {
	key := StateKeyKeys(a)
	value, _ := state.NewHintedValue(keys)
	su, err := state.NewStateV0(key, value, valuehash.RandomSHA256())
	t.NoError(err)

	t.NoError(sp.Set(su))

	ust, found, err := sp.Get(key)
	t.NoError(err)
	t.NotNil(ust)
	t.True(found)

	return su
}

func (t *testTransferOperation) TestSenderNotExist() {
	sp, err := isaac.NewStatePool(t.Storage(nil, nil))
	t.NoError(err)

	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey := NewKey(spk.Publickey(), 100)
	rkey := NewKey(rpk.Publickey(), 100)

	pks := []key.Privatekey{spk, rpk}
	sender, _ := NewAddressFromKeys([]Key{skey})
	receiver, _ := NewAddressFromKeys([]Key{rkey})

	tf := t.newTransfer(sender, receiver, NewAmount(10), pks)

	err = tf.ProcessOperation(
		sp.Get,
		sp.Set,
	)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "keys of sender account does not exist")
}

func (t *testTransferOperation) TestReceiverNotExist() {
	sp, err := isaac.NewStatePool(t.Storage(nil, nil))
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

	err = tf.ProcessOperation(
		sp.Get,
		sp.Set,
	)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "keys of receiver account does not exist")
}

func (t *testTransferOperation) TestInsufficientBalance() {
	sp, err := isaac.NewStatePool(t.Storage(nil, nil))
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

	err = tf.ProcessOperation(
		sp.Get,
		sp.Set,
	)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "invalid amount; under zero")
}

func (t *testTransferOperation) TestSufficientBalance() {
	sp, err := isaac.NewStatePool(t.Storage(nil, nil))
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

	err = tf.ProcessOperation(
		sp.Get,
		sp.Set,
	)
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
	sp, err := isaac.NewStatePool(t.Storage(nil, nil))
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

	err = tf.ProcessOperation(
		sp.Get,
		sp.Set,
	)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "not passed threshold")
}

func (t *testTransferOperation) TestUnknownKey() {
	sp, err := isaac.NewStatePool(t.Storage(nil, nil))
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

	err = tf.ProcessOperation(
		sp.Get,
		sp.Set,
	)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "unknown key found")
}

func TestTransferOperation(t *testing.T) {
	suite.Run(t, new(testTransferOperation))
}
