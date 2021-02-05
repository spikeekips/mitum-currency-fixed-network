package digest

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base/state"
	"golang.org/x/xerrors"
)

func (hd *Handlers) handleCurrencies(w http.ResponseWriter, r *http.Request) {
	if hd.cp == nil {
		hd.notSupported(w, xerrors.Errorf("empty CurrencyPool"))

		return
	}

	if err := loadFromCache(hd.cache, cacheKeyPath(r), w); err != nil {
		hd.Log().Verbose().Err(err).Msg("failed to load cache")
	} else {
		hd.Log().Verbose().Msg("loaded from cache")

		return
	}

	var hal Hal = NewBaseHal(nil, NewHalLink(HandlerPathCurrencies, nil))
	hal = hal.AddLink("currency:{currencyid}", NewHalLink(HandlerPathCurrency, nil).SetTemplated())

	for cid := range hd.cp.Designs() {
		if h, err := hd.combineURL(HandlerPathCurrency, "currencyid", cid.String()); err != nil {
			hd.problemWithError(w, err, http.StatusInternalServerError)

			return
		} else {
			hal = hal.AddLink(fmt.Sprintf("currency:%s", cid), NewHalLink(h, nil))
		}
	}

	hd.writeHal(w, hal, http.StatusOK)
	hd.writeCache(w, cacheKeyPath(r), time.Second*2)
}

func (hd *Handlers) handleCurrency(w http.ResponseWriter, r *http.Request) {
	var cid string
	if s, found := mux.Vars(r)["currencyid"]; !found {
		hd.problemWithError(w, xerrors.Errorf("empty currency id"), http.StatusNotFound)

		return
	} else {
		cid = s
	}

	var de currency.CurrencyDesign
	var st state.State
	if hd.cp == nil {
		hd.notSupported(w, nil)

		return
	} else if i, found := hd.cp.Get(currency.CurrencyID(cid)); !found {
		hd.problemWithError(w, xerrors.Errorf("unknown currency id"), http.StatusNotFound)

		return
	} else if j, found := hd.cp.State(currency.CurrencyID(cid)); !found {
		hd.problemWithError(w, xerrors.Errorf("unknown currency id"), http.StatusNotFound)

		return
	} else {
		de = i
		st = j
	}

	if err := loadFromCache(hd.cache, cacheKeyPath(r), w); err != nil {
		hd.Log().Verbose().Err(err).Msg("failed to load cache")
	} else {
		hd.Log().Verbose().Msg("loaded from cache")

		return
	}

	if hal, err := hd.buildCurrency(de, st); err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	} else {
		hd.writeHal(w, hal, http.StatusOK)
		hd.writeCache(w, cacheKeyPath(r), time.Second*2)
	}
}

func (hd *Handlers) buildCurrency(de currency.CurrencyDesign, st state.State) (Hal, error) {
	var hal Hal

	if h, err := hd.combineURL(HandlerPathCurrency, "currencyid", de.Currency().String()); err != nil {
		return nil, err
	} else {
		hal = NewBaseHal(de, NewHalLink(h, nil))
	}

	hal = hal.AddLink("currency:{currencyid}", NewHalLink(HandlerPathCurrency, nil).SetTemplated())

	if h, err := hd.combineURL(HandlerPathBlockByHeight, "height", st.Height().String()); err != nil {
		return nil, err
	} else {
		hal = hal.AddLink("block", NewHalLink(h, nil))
	}

	for i := range st.Operations() {
		if h, err := hd.combineURL(HandlerPathOperation, "hash", st.Operations()[i].String()); err != nil {
			return nil, err
		} else {
			hal = hal.AddLink("operations", NewHalLink(h, nil))
		}
	}

	return hal, nil
}
