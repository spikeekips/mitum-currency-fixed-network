package currency

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util"
)

type testGenesisAccountOperation struct {
	baseTestOperationProcessor

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

	newAddress, err := NewAddressFromKeys(keys)
	t.NoError(err)

	err = op.Process(sp.Get, sp.Set)
	t.NoError(err)
	t.Equal(2, len(sp.Updates()))

	var ns, nb state.StateUpdater
	for _, st := range sp.Updates() {
		if key := st.Key(); key == StateKeyKeys(newAddress) {
			ns = st
		} else if key == StateKeyBalance(newAddress) {
			nb = st
		}
	}

	ukeys := ns.Value().Interface().(Keys)
	t.Equal(len(keys.Keys()), len(ukeys.Keys()))
	t.Equal(keys.Threshold(), ukeys.Threshold())
	for i := range keys.Keys() {
		t.Equal(keys.Keys()[i].Weight(), ukeys.Keys()[i].Weight())
		t.True(keys.Keys()[i].Key().Equal(ukeys.Keys()[i].Key()))
	}

	t.Equal(amount.String(), nb.Value().Interface())
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

	newAddress, err := NewAddressFromKeys(keys)
	t.NoError(err)

	var ns, nb state.StateUpdater
	for _, st := range sp.Updates() {
		if key := st.Key(); key == StateKeyKeys(newAddress) {
			ns = st
		} else if key == StateKeyBalance(newAddress) {
			nb = st
		}
	}

	ukeys := ns.Value().Interface().(Keys)
	t.Equal(len(keys.Keys()), len(ukeys.Keys()))
	t.Equal(keys.Threshold(), ukeys.Threshold())
	for i := range keys.Keys() {
		t.Equal(keys.Keys()[i].Weight(), ukeys.Keys()[i].Weight())
		t.True(keys.Keys()[i].Key().Equal(ukeys.Keys()[i].Key()))
	}

	t.Equal(amount.String(), nb.Value().Interface())
}

func (t *testGenesisAccountOperation) TestTargetAccountExists() {
	sa, st := t.newAccount(true, NewAmount(3))

	sp, _ := t.statepool(st)

	amount := NewAmount(3333333333333)
	op := t.newOperaton(sa.Keys(), amount)

	err := op.Process(sp.Get, sp.Set)
	t.Contains(err.Error(), "genesis already exists")
}

func TestGenesisAccountOperation(t *testing.T) {
	suite.Run(t, new(testGenesisAccountOperation))
}
