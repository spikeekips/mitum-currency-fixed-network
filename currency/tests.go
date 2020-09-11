// +build test

package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/isaac"
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

	key, err := NewKey(priv.Publickey(), 100)
	if err != nil {
		panic(err)
	}

	keys, err := NewKeys([]Key{key}, 100)
	if err != nil {
		panic(err)
	}

	address, _ := NewAddressFromKeys(keys)

	return &account{Address: address, Priv: priv, Key: key}
}

type baseTest struct { // nolint: unused
	suite.Suite
	isaac.StorageSupportTest
}

func (t *baseTest) SetupSuite() {
	t.StorageSupportTest.SetupSuite()

	_ = t.Encs.AddHinter(key.BTCPublickey{})
	_ = t.Encs.AddHinter(operation.BaseFactSign{})
	_ = t.Encs.AddHinter(Key{})
	_ = t.Encs.AddHinter(Keys{})
	_ = t.Encs.AddHinter(Address(""))
	_ = t.Encs.AddHinter(CreateAccounts{})
	_ = t.Encs.AddHinter(Transfers{})
	_ = t.Encs.AddHinter(KeyUpdaterFact{})
	_ = t.Encs.AddHinter(KeyUpdater{})
	_ = t.Encs.AddHinter(FeeOperationFact{})
	_ = t.Encs.AddHinter(FeeOperation{})
	_ = t.Encs.AddHinter(Account{})
}

func (t *baseTest) newAccount() *account {
	return generateAccount()
}

type baseTestOperationProcessor struct { // nolint: unused
	baseTest
}

func (t *baseTestOperationProcessor) statepool(s ...[]state.State) (*isaac.Statepool, isaac.OperationProcessor) {
	base := map[string]state.State{}
	for _, l := range s {
		for _, st := range l {
			base[st.Key()] = st
		}
	}

	pool, err := isaac.NewStatepoolWithBase(t.Storage(nil, nil), base)
	t.NoError(err)

	opr := (&OperationProcessor{}).New(pool)

	return pool, opr
}

func (t *baseTestOperationProcessor) newAccount(exists bool, amount Amount) (*account, []state.State) {
	ac := t.baseTest.newAccount()

	if !exists {
		return ac, nil
	}

	var st []state.State
	st = append(st,
		t.newStateKeys(ac.Address, ac.Keys()),
		t.newStateBalance(ac.Address, amount),
	)

	return ac, st
}

func (t *baseTestOperationProcessor) newStateBalance(a base.Address, amount Amount) state.State {
	key := StateKeyBalance(a)
	value, _ := state.NewStringValue(amount.String())
	su, err := state.NewStateV0(key, value, base.NilHeight)
	t.NoError(err)

	return su
}

func (t *baseTestOperationProcessor) newStateKeys(a base.Address, keys Keys) state.State {
	key := StateKeyAccount(a)

	ac, err := NewAccount(a, keys)
	t.NoError(err)

	value, _ := state.NewHintedValue(ac)
	su, err := state.NewStateV0(key, value, base.NilHeight)
	t.NoError(err)

	return su
}

func (t *baseTestOperationProcessor) newKey(pub key.Publickey, w uint) Key {
	k, err := NewKey(pub, w)
	if err != nil {
		panic(err)
	}

	return k
}
