package currency

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testGenesisAccountOperation struct {
	suite.Suite
	isaac.StorageSupportTest

	pk        key.Privatekey
	networkID base.NetworkID
}

func (t *testGenesisAccountOperation) SetupSuite() {
	t.StorageSupportTest.SetupSuite()

	t.Encs.AddHinter(key.BTCPublickey{})
	t.Encs.AddHinter(operation.BaseFactSign{})
	t.Encs.AddHinter(Key{})
	t.Encs.AddHinter(Keys{})
	t.Encs.AddHinter(Address(""))
	t.Encs.AddHinter(GenesisAccountFact{})
	t.Encs.AddHinter(GenesisAccount{})

	t.pk = key.MustNewBTCPrivatekey()
	t.networkID = util.UUID().Bytes()
}

func (t *testGenesisAccountOperation) newOperaton(keys Keys, amount Amount) GenesisAccount {
	ga, err := NewGenesisAccount(t.pk, keys, amount, t.networkID)
	t.NoError(err)
	t.NoError(ga.IsValid(t.networkID))

	return ga
}

func (t *testGenesisAccountOperation) TestNew() {
	pk := key.MustNewBTCPrivatekey()
	keys, _ := NewKeys([]Key{NewKey(pk.Publickey(), 100)}, 100)
	amount := NewAmount(3333333333333)

	op := t.newOperaton(keys, amount)

	sp, err := isaac.NewStatepool(t.Storage(nil, nil))
	t.NoError(err)

	newAddress, err := NewAddressFromKeys(keys.Keys())
	t.NoError(err)

	err = op.Process(sp.Get, sp.Set)
	t.NoError(err)
	t.Equal(2, len(sp.Updates()))

	nstate, found, err := sp.Get(StateKeyKeys(newAddress))
	t.NoError(err)
	t.True(found)
	t.NotNil(nstate)

	ukeys := nstate.Value().Interface().(Keys)
	t.Equal(len(keys.Keys()), len(ukeys.Keys()))
	t.Equal(keys.Threshold(), ukeys.Threshold())
	for i := range keys.Keys() {
		t.Equal(keys.Keys()[i].Weight(), ukeys.Keys()[i].Weight())
		t.True(keys.Keys()[i].Key().Equal(ukeys.Keys()[i].Key()))
	}

	nstateBalance, found, err := sp.Get(StateKeyBalance(newAddress))
	t.NoError(err)
	t.True(found)
	t.NotNil(nstateBalance)

	t.Equal(amount.String(), nstateBalance.Value().Interface())
}

func (t *testGenesisAccountOperation) TestMultipleTarget() {
	pk0 := key.MustNewBTCPrivatekey()
	pk1 := key.MustNewBTCPrivatekey()
	keys, _ := NewKeys([]Key{NewKey(pk0.Publickey(), 30), NewKey(pk1.Publickey(), 30)}, 50)
	amount := NewAmount(1333333333333)

	op := t.newOperaton(keys, amount)

	sp, err := isaac.NewStatepool(t.Storage(nil, nil))
	t.NoError(err)

	err = op.Process(sp.Get, sp.Set)
	t.NoError(err)

	newAddress, err := NewAddressFromKeys(keys.Keys())
	t.NoError(err)

	nstate, found, err := sp.Get(StateKeyKeys(newAddress))
	t.NoError(err)
	t.True(found)
	t.NotNil(nstate)

	ukeys := nstate.Value().Interface().(Keys)
	t.Equal(len(keys.Keys()), len(ukeys.Keys()))
	t.Equal(keys.Threshold(), ukeys.Threshold())
	for i := range keys.Keys() {
		t.Equal(keys.Keys()[i].Weight(), ukeys.Keys()[i].Weight())
		t.True(keys.Keys()[i].Key().Equal(ukeys.Keys()[i].Key()))
	}

	nstateBalance, found, err := sp.Get(StateKeyBalance(newAddress))
	t.NoError(err)
	t.True(found)
	t.NotNil(nstateBalance)

	t.Equal(amount.String(), nstateBalance.Value().Interface())
}

func (t *testGenesisAccountOperation) TestTargetAccountExists() {
	pk := key.MustNewBTCPrivatekey()
	keys, _ := NewKeys([]Key{NewKey(pk.Publickey(), 100)}, 100)
	amount := NewAmount(3333333333333)

	op := t.newOperaton(keys, amount)

	sp, err := isaac.NewStatepool(t.Storage(nil, nil))
	t.NoError(err)

	{
		newAddress, err := NewAddressFromKeys(keys.Keys())
		t.NoError(err)
		st, found, err := sp.Get(StateKeyBalance(newAddress))
		t.NoError(err)
		t.False(found)
		sp.Set(valuehash.RandomSHA256(), st)
	}

	err = op.Process(sp.Get, sp.Set)
	t.Contains(err.Error(), "balance of genesis already exists")
}

func TestGenesisAccountOperation(t *testing.T) {
	suite.Run(t, new(testGenesisAccountOperation))
}
