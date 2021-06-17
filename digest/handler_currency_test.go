// +build mongodb

package digest

import (
	"io"
	"testing"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/state"
	"github.com/stretchr/testify/suite"
)

type testHandlerCurrency struct {
	baseTestHandlers
}

func (t *testHandlerCurrency) TestCurrencies() {
	cp := currency.NewCurrencyPool()

	var de currency.CurrencyDesign
	{

		big := currency.NewBig(33)
		cid := currency.CurrencyID("BLK")

		de = currency.NewCurrencyDesign(
			currency.MustNewAmount(big, cid),
			currency.NewTestAddress(),
			currency.NewCurrencyPolicy(
				currency.NewBig(1),
				currency.NewFixedFeeer(
					currency.NewTestAddress(),
					currency.NewBig(99),
				),
			),
		)

		st, err := state.NewStateV0(currency.StateKeyCurrencyDesign(de.Currency()), nil, base.Height(33))
		t.NoError(err)

		nst, err := currency.SetStateCurrencyDesignValue(st, de)
		t.NoError(err)

		cp.Set(nst)
	}

	handlers := NewHandlers(t.networkID, t.Encs, t.JSONEnc, nil, DummyCache{}, cp)
	t.NoError(handlers.Initialize())

	self, err := handlers.router.Get(HandlerPathCurrencies).URL()
	t.NoError(err)

	w := t.requestOK(handlers, "GET", self.Path, nil)

	b, err := io.ReadAll(w.Result().Body)
	t.NoError(err)

	hal := t.loadHal(b)

	t.Equal(self.String(), hal.Links()["self"].Href())

	currencyLink, err := handlers.router.Get(HandlerPathCurrency).URLPath("currencyid", de.Currency().String())
	t.NoError(err)
	t.Equal(currencyLink.Path, hal.Links()["currency:"+de.Currency().String()].Href())
}

func (t *testHandlerCurrency) TestCurrency() {
	cp := currency.NewCurrencyPool()

	var de currency.CurrencyDesign
	{

		big := currency.NewBig(33)
		cid := currency.CurrencyID("BLK")

		de = currency.NewCurrencyDesign(
			currency.MustNewAmount(big, cid),
			currency.NewTestAddress(),
			currency.NewCurrencyPolicy(
				currency.NewBig(1),
				currency.NewFixedFeeer(
					currency.NewTestAddress(),
					currency.NewBig(99),
				),
			),
		)

		st, err := state.NewStateV0(currency.StateKeyCurrencyDesign(de.Currency()), nil, base.Height(33))
		t.NoError(err)

		nst, err := currency.SetStateCurrencyDesignValue(st, de)
		t.NoError(err)

		cp.Set(nst)
	}

	handlers := NewHandlers(t.networkID, t.Encs, t.JSONEnc, nil, DummyCache{}, cp)
	t.NoError(handlers.Initialize())

	self, err := handlers.router.Get(HandlerPathCurrency).URLPath("currencyid", de.Currency().String())
	t.NoError(err)

	w := t.requestOK(handlers, "GET", self.Path, nil)

	b, err := io.ReadAll(w.Result().Body)
	t.NoError(err)

	hal := t.loadHal(b)

	t.Equal(self.String(), hal.Links()["self"].Href())

	hinter, err := t.JSONEnc.Decode(hal.RawInterface())
	t.NoError(err)
	ude, ok := hinter.(currency.CurrencyDesign)
	t.True(ok)

	t.compareCurrencyDesign(de, ude)
}

func TestHandlerCurrency(t *testing.T) {
	suite.Run(t, new(testHandlerCurrency))
}
