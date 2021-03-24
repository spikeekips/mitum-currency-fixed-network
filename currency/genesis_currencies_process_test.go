package currency

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
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

	t.Encs.AddHinter(key.BTCPublickey{})
	t.Encs.AddHinter(operation.BaseFactSign{})
	t.Encs.AddHinter(Key{})
	t.Encs.AddHinter(Keys{})
	t.Encs.AddHinter(Address(""))
	t.Encs.AddHinter(GenesisCurrenciesFact{})
	t.Encs.AddHinter(GenesisCurrencies{})
	t.Encs.AddHinter(Account{})
	t.Encs.AddHinter(Amount{})
	t.Encs.AddHinter(CurrencyDesign{})
	t.Encs.AddHinter(CurrencyPolicy{})

	t.pk = key.MustNewBTCPrivatekey()
	t.networkID = util.UUID().Bytes()
}

func (t *testGenesisCurrenciesOperation) newOperaton(keys Keys, cs []CurrencyDesign) GenesisCurrencies {
	gc, err := NewGenesisCurrencies(t.pk, keys, cs, t.networkID)
	t.NoError(err)
	t.NoError(gc.IsValid(t.networkID))

	return gc
}

func (t *testGenesisCurrenciesOperation) genesisCurrency(cid string, amount int64) CurrencyDesign {
	return NewCurrencyDesign(MustNewAmount(NewBig(amount), CurrencyID(cid)), nil, NewCurrencyPolicy(ZeroBig, NewNilFeeer()))
}

func (t *testGenesisCurrenciesOperation) TestNew() {
	pk := key.MustNewBTCPrivatekey()
	keys, _ := NewKeys([]Key{t.newKey(pk.Publickey(), 100)}, 100)
	cs := []CurrencyDesign{
		t.genesisCurrency("FIND*ME", 44),
		t.genesisCurrency("SHOW_ME", 33),
	}

	op := t.newOperaton(keys, cs)

	sp, err := storage.NewStatepool(t.Database(nil, nil))
	t.NoError(err)

	newAddress, err := NewAddressFromKeys(keys)
	t.NoError(err)

	err = op.Process(sp.Get, sp.Set)
	t.NoError(err)
	t.Equal(5, len(sp.Updates()))

	var ns state.State
	var nb []state.State
	dts := map[CurrencyID]CurrencyDesign{}
	for _, st := range sp.Updates() {
		if key := st.Key(); key == StateKeyAccount(newAddress) {
			ns = st.GetState()
		} else if IsStateBalanceKey(key) {
			nb = append(nb, st.GetState())
		} else if IsStateCurrencyDesignKey(key) {
			i, err := StateCurrencyDesignValue(st.GetState())
			t.NoError(err)
			dts[i.Currency()] = i
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

		t.compareCurrencyDesign(a, b)
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
