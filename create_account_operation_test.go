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

type testCreateAccountOperation struct {
	suite.Suite
	isaac.StorageSupportTest
}

func (t *testCreateAccountOperation) SetupSuite() {
	t.StorageSupportTest.SetupSuite()

	t.Encs.AddHinter(key.BTCPublickey{})
	t.Encs.AddHinter(operation.BaseFactSign{})
	t.Encs.AddHinter(Key{})
	t.Encs.AddHinter(Keys{})
	t.Encs.AddHinter(Address(""))
	t.Encs.AddHinter(Transfer{})
}

func (t *testCreateAccountOperation) newOperation(sender Address, amount Amount, keys Keys, pks []key.Privatekey) CreateAccount {
	token := util.UUID().Bytes()
	fact := NewCreateAccountFact(token, sender, keys, amount)

	var fs []operation.FactSign
	for _, pk := range pks {
		sig, err := operation.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
	}

	ca, err := NewCreateAccount(fact, fs, "")
	t.NoError(err)

	t.NoError(ca.IsValid(nil))

	return ca
}

func (t *testCreateAccountOperation) newStateBalance(a Address, amount Amount, sp *isaac.Statepool) state.StateUpdater {
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

func (t *testCreateAccountOperation) newStateKeys(a Address, keys Keys, sp *isaac.Statepool) state.StateUpdater {
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

func (t *testCreateAccountOperation) TestSufficientBalance() {
	sp, err := isaac.NewStatepool(t.Storage(nil, nil))
	t.NoError(err)

	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey := NewKey(spk.Publickey(), 100)
	skeys, _ := NewKeys([]Key{skey}, 100)

	rkey := NewKey(rpk.Publickey(), 100)
	rkeys, _ := NewKeys([]Key{rkey}, 100)

	sender, _ := NewAddressFromKeys([]Key{skey})
	receiver, _ := NewAddressFromKeys([]Key{rkey})

	// set sender state
	senderBalance := NewAmount(33)
	amount := NewAmount(10)

	_ = t.newStateBalance(sender, senderBalance, sp)
	_ = t.newStateKeys(sender, skeys, sp)

	ca := t.newOperation(sender, amount, rkeys, []key.Privatekey{spk})

	err = ca.ProcessOperation(
		sp.Get,
		sp.Set,
	)
	t.NoError(err)

	// checking value
	sstate, found, err := sp.Get(StateKeyBalance(sender))
	t.NoError(err)
	t.True(found)
	t.NotNil(sstate)

	rstateBalance, found, err := sp.Get(StateKeyBalance(receiver))
	t.NoError(err)
	t.True(found)
	t.NotNil(rstateBalance)

	t.Equal(sstate.Value().Interface(), "23")
	t.Equal(rstateBalance.Value().Interface(), amount.String())

	rstate, found, err := sp.Get(StateKeyKeys(receiver))
	t.NoError(err)
	t.True(found)
	t.NotNil(rstate)

	ukeys := rstate.Value().Interface().(Keys)
	t.Equal(len(rkeys.Keys()), len(ukeys.Keys()))
	t.Equal(rkeys.Threshold(), ukeys.Threshold())
	for i := range rkeys.Keys() {
		t.Equal(rkeys.Keys()[i].Weight(), ukeys.Keys()[i].Weight())
		t.True(rkeys.Keys()[i].Key().Equal(ukeys.Keys()[i].Key()))
	}
}

func (t *testCreateAccountOperation) TestSenderKeysNotExist() {
	sp, err := isaac.NewStatepool(t.Storage(nil, nil))
	t.NoError(err)

	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey := NewKey(spk.Publickey(), 100)

	rkey := NewKey(rpk.Publickey(), 100)
	rkeys, _ := NewKeys([]Key{rkey}, 100)

	sender, _ := NewAddressFromKeys([]Key{skey})

	amount := NewAmount(10)
	ca := t.newOperation(sender, amount, rkeys, []key.Privatekey{spk})

	err = ca.ProcessOperation(
		sp.Get,
		sp.Set,
	)

	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "keys of sender does not exist")
}

func (t *testCreateAccountOperation) TestSenderBalanceNotExist() {
	sp, err := isaac.NewStatepool(t.Storage(nil, nil))
	t.NoError(err)

	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey := NewKey(spk.Publickey(), 100)
	skeys, _ := NewKeys([]Key{skey}, 100)

	rkey := NewKey(rpk.Publickey(), 100)
	rkeys, _ := NewKeys([]Key{rkey}, 100)

	sender, _ := NewAddressFromKeys([]Key{skey})

	_ = t.newStateKeys(sender, skeys, sp)

	amount := NewAmount(10)
	ca := t.newOperation(sender, amount, rkeys, []key.Privatekey{spk})

	err = ca.ProcessOperation(
		sp.Get,
		sp.Set,
	)

	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "balance of sender does not exist")
}

func (t *testCreateAccountOperation) TestReceiverExists() {
	sp, err := isaac.NewStatepool(t.Storage(nil, nil))
	t.NoError(err)

	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey := NewKey(spk.Publickey(), 100)
	skeys, _ := NewKeys([]Key{skey}, 100)

	rkey := NewKey(rpk.Publickey(), 100)
	rkeys, _ := NewKeys([]Key{rkey}, 100)

	sender, _ := NewAddressFromKeys([]Key{skey})
	receiver, _ := NewAddressFromKeys([]Key{rkey})

	// set sender state
	senderBalance := NewAmount(33)
	amount := NewAmount(10)

	_ = t.newStateBalance(sender, senderBalance, sp)
	_ = t.newStateKeys(sender, skeys, sp)

	_ = t.newStateBalance(receiver, NewAmount(3), sp)
	_ = t.newStateKeys(receiver, rkeys, sp)

	ca := t.newOperation(sender, amount, rkeys, []key.Privatekey{spk})

	err = ca.ProcessOperation(
		sp.Get,
		sp.Set,
	)
	t.Contains(err.Error(), "keys of target already exists")
}

func (t *testCreateAccountOperation) TestInsufficientBalance() {
	sp, err := isaac.NewStatepool(t.Storage(nil, nil))
	t.NoError(err)

	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey := NewKey(spk.Publickey(), 100)
	skeys, _ := NewKeys([]Key{skey}, 100)

	rkey := NewKey(rpk.Publickey(), 100)
	rkeys, _ := NewKeys([]Key{rkey}, 100)

	sender, _ := NewAddressFromKeys([]Key{skey})

	// set sender state
	amount := NewAmount(10)
	senderBalance := amount.Sub(NewAmount(3))

	_ = t.newStateBalance(sender, senderBalance, sp)
	_ = t.newStateKeys(sender, skeys, sp)

	ca := t.newOperation(sender, amount, rkeys, []key.Privatekey{spk})

	err = ca.ProcessOperation(
		sp.Get,
		sp.Set,
	)
	t.Contains(err.Error(), "invalid amount; under zero")
}

func TestCreateAccountOperation(t *testing.T) {
	suite.Run(t, new(testCreateAccountOperation))
}
