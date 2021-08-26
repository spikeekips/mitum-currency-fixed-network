package digest

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base/state"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/util"
)

func (hd *Handlers) handleCurrencies(w http.ResponseWriter, r *http.Request) {
	if hd.cp == nil {
		HTTP2NotSupported(w, errors.Errorf("empty CurrencyPool"))

		return
	}

	cachekey := CacheKeyPath(r)
	if err := LoadFromCache(hd.cache, cachekey, w); err == nil {
		return
	}

	if v, err, shared := hd.rg.Do(cachekey, func() (interface{}, error) {
		return hd.handleCurrenciesInGroup()
	}); err != nil {
		HTTP2HandleError(w, err)
	} else {
		HTTP2WriteHalBytes(hd.enc, w, v.([]byte), http.StatusOK)

		if !shared {
			HTTP2WriteCache(w, cachekey, time.Second*3)
		}
	}
}

func (hd *Handlers) handleCurrenciesInGroup() ([]byte, error) {
	var hal Hal = NewBaseHal(nil, NewHalLink(HandlerPathCurrencies, nil))
	hal = hal.AddLink("currency:{currencyid}", NewHalLink(HandlerPathCurrency, nil).SetTemplated())

	for cid := range hd.cp.Designs() {
		h, err := hd.combineURL(HandlerPathCurrency, "currencyid", cid.String())
		if err != nil {
			return nil, err
		}
		hal = hal.AddLink(fmt.Sprintf("currency:%s", cid), NewHalLink(h, nil))
	}

	return hd.enc.Marshal(hal)
}

func (hd *Handlers) handleCurrency(w http.ResponseWriter, r *http.Request) {
	cachekey := CacheKeyPath(r)
	if err := LoadFromCache(hd.cache, cachekey, w); err == nil {
		return
	}

	var cid string
	s, found := mux.Vars(r)["currencyid"]
	if !found {
		HTTP2ProblemWithError(w, errors.Errorf("empty currency id"), http.StatusNotFound)

		return
	}

	s = strings.TrimSpace(s)
	if len(s) < 1 {
		HTTP2ProblemWithError(w, errors.Errorf("empty currency id"), http.StatusBadRequest)

		return
	}
	cid = s

	if v, err, shared := hd.rg.Do(cachekey, func() (interface{}, error) {
		return hd.handleCurrencyInGroup(cid)
	}); err != nil {
		HTTP2HandleError(w, err)
	} else {
		HTTP2WriteHalBytes(hd.enc, w, v.([]byte), http.StatusOK)

		if !shared {
			HTTP2WriteCache(w, cachekey, time.Second*3)
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

	i, err := hd.buildCurrency(de, st)
	if err != nil {
		return nil, err
	}
	return hd.enc.Marshal(i)
}

func (hd *Handlers) buildCurrency(de currency.CurrencyDesign, st state.State) (Hal, error) {
	h, err := hd.combineURL(HandlerPathCurrency, "currencyid", de.Currency().String())
	if err != nil {
		return nil, err
	}

	var hal Hal
	hal = NewBaseHal(de, NewHalLink(h, nil))

	hal = hal.AddLink("currency:{currencyid}", NewHalLink(HandlerPathCurrency, nil).SetTemplated())

	h, err = hd.combineURL(HandlerPathBlockByHeight, "height", st.Height().String())
	if err != nil {
		return nil, err
	}
	hal = hal.AddLink("block", NewHalLink(h, nil))

	for i := range st.Operations() {
		h, err := hd.combineURL(HandlerPathOperation, "hash", st.Operations()[i].String())
		if err != nil {
			return nil, err
		}
		hal = hal.AddLink("operations", NewHalLink(h, nil))
	}

	return hal, nil
}
