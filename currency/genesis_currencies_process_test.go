package currency

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

type testGenesisCurrenciesOperation struct {
	baseTestOperationProcessor

	pk        key.Privatekey
	networkID base.NetworkID
}

func (t *testGenesisCurrenciesOperation) SetupSuite() {
	t.StorageSupportTest.SetupSuite()

	t.Encs.TestAddHinter(key.BasePublickey{})
	t.Encs.TestAddHinter(base.BaseFactSign{})
	t.Encs.TestAddHinter(AccountKeyHinter)
	t.Encs.TestAddHinter(AccountKeysHinter)
	t.Encs.TestAddHinter(AddressHinter)
	t.Encs.TestAddHinter(GenesisCurrenciesFactHinter)
	t.Encs.TestAddHinter(GenesisCurrenciesHinter)
	t.Encs.TestAddHinter(AccountHinter)
	t.Encs.TestAddHinter(AmountHinter)
	t.Encs.TestAddHinter(CurrencyDesignHinter)
	t.Encs.TestAddHinter(CurrencyPolicyHinter)

	t.pk = key.NewBasePrivatekey()
	t.networkID = util.UUID().Bytes()
}

func (t *testGenesisCurrenciesOperation) newOperaton(keys AccountKeys, cs []CurrencyDesign) GenesisCurrencies {
	gc, err := NewGenesisCurrencies(t.pk, keys, cs, t.networkID)
	t.NoError(err)
	t.NoError(gc.IsValid(t.networkID))

	return gc
}

func (t *testGenesisCurrenciesOperation) genesisCurrency(cid string, amount int64) CurrencyDesign {
	return NewCurrencyDesign(MustNewAmount(NewBig(amount), CurrencyID(cid)), nil, NewCurrencyPolicy(ZeroBig, NewNilFeeer()))
}

func (t *testGenesisCurrenciesOperation) TestNew() {
	pk := key.NewBasePrivatekey()
	keys, _ := NewBaseAccountKeys([]AccountKey{t.newKey(pk.Publickey(), 100)}, 100)
	cs := []CurrencyDesign{
		t.genesisCurrency("FIND*ME", 44),
		t.genesisCurrency("SHOW_ME", 33),
	}

	op := t.newOperaton(keys, cs)
	fact := op.Fact().(GenesisCurrenciesFact)

	sp, err := storage.NewStatepool(t.Database(nil, nil))
	t.NoError(err)

	newAddress, err := NewAddressFromKeys(keys)
	t.NoError(err)
	{
		fa, err := fact.Address()
		t.NoError(err)
		t.True(newAddress.Equal(fa))
	}

	err = op.Process(sp.Get, sp.Set)
	t.NoError(err)
	t.Equal(9, len(sp.Updates()))

	var ns state.State
	var nb []state.State
	zast := map[CurrencyID]state.State{}
	zbst := map[CurrencyID]state.State{}
	dts := map[CurrencyID]CurrencyDesign{}
	for _, st := range sp.Updates() {
		key := st.Key()
		switch {
		case key == StateKeyAccount(newAddress):
			ns = st.GetState()
		case IsStateCurrencyDesignKey(key):
			i, err := StateCurrencyDesignValue(st.GetState())
			t.NoError(err)
			dts[i.Currency()] = i
		}

		for i := range cs {
			cid := cs[i].Currency()
			zac, err := ZeroAccount(cid)
			t.NoError(err)

			switch {
			case key == StateKeyAccount(zac.Address()):
				zast[cid] = st.GetState()
			case key == StateKeyBalance(newAddress, cid):
				nb = append(nb, st.GetState())
			case key == StateKeyBalance(zac.Address(), cid):
				zbst[cid] = st.GetState()
			}
		}
	}

	sort.Slice(nb, func(i, j int) bool {
		return strings.Compare(nb[i].Key(), nb[j].Key()) < 0
	})

	ac := ns.Value().Interface().(Account)
	ukeys := ac.Keys()
	t.Equal(len(keys.Keys()), len(ukeys.Keys()))
	t.Equal(keys.Threshold(), ukeys.Threshold())
	for i := range keys.Keys() {
		t.Equal(keys.Keys()[i].Weight(), ukeys.Keys()[i].Weight())
		t.True(keys.Keys()[i].Key().Equal(ukeys.Keys()[i].Key()))
	}

	t.Equal(len(cs), len(nb))

	t.Equal(cs[0].Amount, nb[0].Value().Interface())
	t.Equal(cs[1].Amount, nb[1].Value().Interface())

	t.Equal(len(cs), len(dts))

	for _, a := range cs {
		b, found := dts[a.Currency()]
		t.True(found)
		t.NotNil(b.GenesisAccount())
		t.True(b.GenesisAccount().Equal(newAddress))

		t.compareCurrencyDesign(a, b)
	}

	// NOTE zero
	for i := range cs {
		cid := cs[i].Currency()
		zac, err := ZeroAccount(cid)
		t.NoError(err)

		ast, found := zast[cid]
		t.True(found)
		t.NotNil(ast)

		bst, found := zbst[cid]
		t.True(found)
		t.NotNil(bst)

		ac := ast.Value().Interface().(Account)
		t.True(zac.Address().Equal(ac.Address()))

		b := bst.Value().Interface().(Amount)
		t.True(b.Big().IsZero())
	}
}

func (t *testGenesisCurrenciesOperation) TestTargetAccountExists() {
	sa, st := t.newAccount(true, nil)

	sp, _ := t.statepool(st)

	op := t.newOperaton(sa.Keys(), []CurrencyDesign{t.genesisCurrency("SHOW_ME", 33)})

	err := op.Process(sp.Get, sp.Set)
	t.Contains(err.Error(), "genesis already exists")
}

func TestGenesisCurrenciesOperation(t *testing.T) {
	suite.Run(t, new(testGenesisCurrenciesOperation))
}
