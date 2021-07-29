// +build mongodb

package digest

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/state"
	"github.com/stretchr/testify/suite"
)

type testCurrency struct {
	baseTest
}

func (t *testCurrency) newCurrencyDesign(i int) currency.CurrencyDesign {
	cid := currency.CurrencyID(fmt.Sprintf("BLK.%d", i))

	var fee currency.Big
	for {
		if b := t.randomBig(); b.OverZero() {
			fee = b
			break
		}
	}

	var am currency.Big
	for {
		am = t.randomBig()
		if am.Cmp(big.NewInt(0)) > 0 {
			break
		}
	}

	de := currency.NewCurrencyDesign(
		currency.MustNewAmount(t.randomBig(), cid),
		currency.NewTestAddress(),
		currency.NewCurrencyPolicy(
			am,
			currency.NewFixedFeeer(currency.NewTestAddress(), fee),
		),
	)

	t.NoError(de.IsValid(nil))

	return de
}

func (t *testCurrency) newCurrencyDesignState(de currency.CurrencyDesign, height base.Height) state.State {
	st, err := state.NewStateV0(currency.StateKeyCurrencyDesign(de.Currency()), nil, height)
	t.NoError(err)

	nst, err := currency.SetStateCurrencyDesignValue(st, de)
	t.NoError(err)

	return nst
}

func (t *testCurrency) newHeight(excude base.Height) base.Height {
	for {
		h := base.Height(t.randomBig().Int64() % 300)
		if h != excude {
			return h
		}
	}
}

func (t *testCurrency) TestLoad() {
	des := make([]currency.CurrencyDesign, 3)

	mst := t.MongodbDatabase()

	var sts []state.State
	cids := map[currency.CurrencyID]currency.CurrencyDesign{}
	for i := range des {
		h0 := base.Height(t.randomBig().Int64() % 300)
		de0 := t.newCurrencyDesign(i)
		st0 := t.newCurrencyDesignState(de0, h0)

		h1 := t.newHeight(h0)
		de1 := t.newCurrencyDesign(i)
		st1 := t.newCurrencyDesignState(de1, h1)

		var last currency.CurrencyDesign
		if h0 > h1 {
			last = de0
		} else {
			last = de1
		}
		cids[last.Currency()] = last

		sts = append(sts, st0, st1)
	}

	for _, st := range sts {
		t.NoError(mst.NewState(st))
	}

	cp := currency.NewCurrencyPool()

	t.NoError(LoadCurrenciesFromDatabase(mst, base.NilHeight, func(sta state.State) (bool, error) {
		t.NoError(cp.Set(sta))

		return true, nil
	}))

	t.Equal(len(cids), len(cp.CIDs()))

	for cid := range cids {
		ade := cids[cid]
		bde, found := cp.Get(cid)

		t.True(found)

		t.compareCurrencyDesign(ade, bde)
	}
}

func TestCurrency(t *testing.T) {
	suite.Run(t, new(testCurrency))
}
