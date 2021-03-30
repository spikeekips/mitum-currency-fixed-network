package digest

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base/state"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
)

func (hd *Handlers) handleCurrencies(w http.ResponseWriter, r *http.Request) {
	if hd.cp == nil {
		hd.notSupported(w, xerrors.Errorf("empty CurrencyPool"))

		return
	}

	cachekey := cacheKeyPath(r)
	if err := loadFromCache(hd.cache, cachekey, w); err != nil {
		hd.Log().Verbose().Err(err).Msg("failed to load cache")
	} else {
		hd.Log().Verbose().Msg("loaded from cache")
		return
	}

	if v, err, shared := hd.rg.Do(cachekey, func() (interface{}, error) {
		return hd.handleCurrenciesInGroup()
	}); err != nil {
		hd.handleError(w, err)
	} else {
		hd.writeHalBytes(w, v.([]byte), http.StatusOK)

		if !shared {
			hd.writeCache(w, cachekey, time.Second*3)
		}
	}
}

func (hd *Handlers) handleCurrenciesInGroup() ([]byte, error) {
	var hal Hal = NewBaseHal(nil, NewHalLink(HandlerPathCurrencies, nil))
	hal = hal.AddLink("currency:{currencyid}", NewHalLink(HandlerPathCurrency, nil).SetTemplated())

	for cid := range hd.cp.Designs() {
		if h, err := hd.combineURL(HandlerPathCurrency, "currencyid", cid.String()); err != nil {
			return nil, err
		} else {
			hal = hal.AddLink(fmt.Sprintf("currency:%s", cid), NewHalLink(h, nil))
		}
	}

	return hd.enc.Marshal(hal)
}

func (hd *Handlers) handleCurrency(w http.ResponseWriter, r *http.Request) {
	cachekey := cacheKeyPath(r)
	if err := loadFromCache(hd.cache, cachekey, w); err != nil {
		hd.Log().Verbose().Err(err).Msg("failed to load cache")
	} else {
		hd.Log().Verbose().Msg("loaded from cache")
		return
	}

	var cid string
	if s, found := mux.Vars(r)["currencyid"]; !found {
		hd.problemWithError(w, xerrors.Errorf("empty currency id"), http.StatusNotFound)

		return
	} else {
		s = strings.TrimSpace(s)
		if len(s) < 1 {
			hd.problemWithError(w, xerrors.Errorf("empty currency id"), http.StatusBadRequest)

			return
		}
		cid = s
	}

	if v, err, shared := hd.rg.Do(cachekey, func() (interface{}, error) {
		return hd.handleCurrencyInGroup(cid)
	}); err != nil {
		hd.handleError(w, err)
	} else {
		hd.writeHalBytes(w, v.([]byte), http.StatusOK)

		if !shared {
			hd.writeCache(w, cachekey, time.Second*3)
		}
	}
}

func (hd *Handlers) handleCurrencyInGroup(cid string) ([]byte, error) {
	var de currency.CurrencyDesign
	var st state.State
	if hd.cp == nil {
		return nil, quicnetwork.NotSupportedErorr.Errorf("missing currency pool")
	} else if i, found := hd.cp.Get(currency.CurrencyID(cid)); !found {
		return nil, util.NotFoundError.Errorf("unknown currency id, %q", cid)
	} else if j, found := hd.cp.State(currency.CurrencyID(cid)); !found {
		return nil, util.NotFoundError.Errorf("unknown currency id, %q", cid)
	} else {
		de = i
		st = j
	}

	if i, err := hd.buildCurrency(de, st); err != nil {
		return nil, err
	} else {
		return hd.enc.Marshal(i)
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
