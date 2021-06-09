// +build test

package currency

import (
	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/storage"
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
	cid CurrencyID
}

func (t *baseTest) SetupSuite() {
	t.StorageSupportTest.SetupSuite()

	_ = t.Encs.TestAddHinter(key.BTCPublickey{})
	_ = t.Encs.TestAddHinter(operation.BaseFactSign{})
	_ = t.Encs.TestAddHinter(Key{})
	_ = t.Encs.TestAddHinter(Keys{})
	_ = t.Encs.TestAddHinter(Address(""))
	_ = t.Encs.TestAddHinter(CreateAccounts{})
	_ = t.Encs.TestAddHinter(Transfers{})
	_ = t.Encs.TestAddHinter(KeyUpdaterFact{})
	_ = t.Encs.TestAddHinter(KeyUpdater{})
	_ = t.Encs.TestAddHinter(FeeOperationFact{})
	_ = t.Encs.TestAddHinter(FeeOperation{})
	_ = t.Encs.TestAddHinter(Account{})
	_ = t.Encs.TestAddHinter(CurrencyDesign{})
	_ = t.Encs.TestAddHinter(CurrencyPolicyUpdaterFact{})
	_ = t.Encs.TestAddHinter(CurrencyPolicyUpdater{})
	_ = t.Encs.TestAddHinter(CurrencyPolicy{})

	t.cid = CurrencyID("SEEME")
}

func (t *baseTest) newAccount() *account {
	return generateAccount()
}

func (t *baseTest) currencyDesign(big Big, cid CurrencyID) CurrencyDesign {
	return NewCurrencyDesign(NewAmount(big, cid), NewTestAddress(), NewCurrencyPolicy(ZeroBig, NewNilFeeer()))
}

func (t *baseTest) compareCurrencyDesign(a, b CurrencyDesign) {
	t.True(a.Amount.Equal(b.Amount))
	if a.GenesisAccount() != nil {
		t.True(a.GenesisAccount().Equal(a.GenesisAccount()))
	}
	t.Equal(a.Policy(), b.Policy())
}

type baseTestOperationProcessor struct { // nolint: unused
	baseTest
}

func (t *baseTestOperationProcessor) statepool(s ...[]state.State) (*storage.Statepool, prprocessor.OperationProcessor) {
	base := map[string]state.State{}
	for _, l := range s {
		for _, st := range l {
			base[st.Key()] = st
		}
	}

	pool, err := storage.NewStatepoolWithBase(t.Database(nil, nil), base)
	t.NoError(err)

	opr := (NewOperationProcessor(nil)).New(pool)

	return pool, opr
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

func (t *baseTestOperationProcessor) newAccount(exists bool, amounts []Amount) (*account, []state.State) {
	ac := t.baseTest.newAccount()

	if !exists {
		return ac, nil
	}

	var sts []state.State
	sts = append(sts, t.newStateKeys(ac.Address, ac.Keys()))

	for _, am := range amounts {
		sts = append(sts, t.newStateAmount(ac.Address, am))
	}

	return ac, sts
}

func (t *baseTestOperationProcessor) newStateAmount(a base.Address, amount Amount) state.State {
	key := StateKeyBalance(a, amount.Currency())
	value, _ := state.NewHintedValue(amount)
	su, err := state.NewStateV0(key, value, base.NilHeight)
	t.NoError(err)

	return su
}

func (t *baseTestOperationProcessor) newStateBalance(a base.Address, big Big, cid CurrencyID) state.State {
	key := StateKeyBalance(a, cid)
	value, _ := state.NewHintedValue(NewAmount(big, cid))
	su, err := state.NewStateV0(key, value, base.NilHeight)
	t.NoError(err)

	return su
}

func (t *baseTestOperationProcessor) newCurrencyDesignState(cid CurrencyID, big Big, genesisAccount base.Address, feeer Feeer) state.State {
	de := NewCurrencyDesign(NewAmount(big, cid), genesisAccount, NewCurrencyPolicy(ZeroBig, feeer))

	st, err := state.NewStateV0(StateKeyCurrencyDesign(cid), nil, base.NilHeight)
	t.NoError(err)

	nst, err := SetStateCurrencyDesignValue(st, de)
	t.NoError(err)

	return nst
}

func NewTestAddress() base.Address {
	k, err := NewKey(key.MustNewBTCPrivatekey().Publickey(), 100)
	if err != nil {
		panic(err)
	}

	keys, err := NewKeys([]Key{k}, 100)
	if err != nil {
		panic(err)
	}

	a, err := NewAddressFromKeys(keys)
	if err != nil {
		panic(err)
	}

	return a
}
