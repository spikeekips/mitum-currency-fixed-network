// +build test

package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type account struct { // nolint: unused
	Address base.Address
	Priv    key.Privatekey
	Key     Key
}

func (ac *account) Privs() []key.Privatekey {
	return []key.Privatekey{ac.Priv}
}

func (ac *account) Keys() Keys {
	keys, _ := NewKeys([]Key{ac.Key}, 100)

	return keys
}

func generateAccount() *account { // nolint: unused
	priv := key.MustNewBTCPrivatekey()

	key := NewKey(priv.Publickey(), 100)
	address, _ := NewAddressFromKeys([]Key{key})

	return &account{Address: address, Priv: priv, Key: key}
}

type baseTestOperationProcessor struct { // nolint: unused
	suite.Suite
	isaac.StorageSupportTest
	pool *isaac.Statepool
	opr  isaac.OperationProcessor
}

func (t *baseTestOperationProcessor) SetupSuite() {
	t.StorageSupportTest.SetupSuite()

	_ = t.Encs.AddHinter(key.BTCPublickey{})
	_ = t.Encs.AddHinter(operation.BaseFactSign{})
	_ = t.Encs.AddHinter(Key{})
	_ = t.Encs.AddHinter(Keys{})
	_ = t.Encs.AddHinter(Address(""))
	_ = t.Encs.AddHinter(CreateAccount{})
	_ = t.Encs.AddHinter(Transfer{})
}

func (t *baseTestOperationProcessor) SetupTest() {
	pool, err := isaac.NewStatepool(t.Storage(nil, nil))
	t.NoError(err)
	t.pool = pool

	opr := &OperationProcessor{}
	t.opr = opr.New(t.pool)
}

func (t *baseTestOperationProcessor) newAccount(exists bool, amount Amount) *account {
	ac := generateAccount()

	if !exists {
		return ac
	}

	_ = t.newStateKeys(ac.Address, ac.Keys())
	_ = t.newStateBalance(ac.Address, amount)

	return ac
}

func (t *baseTestOperationProcessor) newStateBalance(a base.Address, amount Amount) state.StateUpdater {
	key := StateKeyBalance(a)
	value, _ := state.NewStringValue(amount.String())
	su, err := state.NewStateV0Updater(key, value, valuehash.RandomSHA256())
	t.NoError(err)

	t.NoError(t.pool.Set(valuehash.RandomSHA256(), su))

	ust, found, err := t.pool.Get(key)
	t.NoError(err)
	t.NotNil(ust)
	t.True(found)

	return su
}

func (t *baseTestOperationProcessor) newStateKeys(a base.Address, keys Keys) state.StateUpdater {
	key := StateKeyKeys(a)
	value, _ := state.NewHintedValue(keys)
	su, err := state.NewStateV0Updater(key, value, valuehash.RandomSHA256())
	t.NoError(err)

	t.NoError(t.pool.Set(valuehash.RandomSHA256(), su))

	ust, found, err := t.pool.Get(key)
	t.NoError(err)
	t.NotNil(ust)
	t.True(found)

	return su
}
