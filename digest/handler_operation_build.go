package digest

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/util/hint"
)

var factTypesByHint = map[string]hint.Hinter{
	"create-accounts":   currency.CreateAccountsHinter,
	"key-updater":       currency.KeyUpdaterHinter,
	"transfers":         currency.TransfersHinter,
	"currency-register": currency.CurrencyRegisterHinter,
}

func (hd *Handlers) handleOperationBuild(w http.ResponseWriter, r *http.Request) {
	if err := LoadFromCache(hd.cache, CacheKeyPath(r), w); err == nil {
		return
	}

	h, err := hd.combineURL(HandlerPathOperationBuild)
	if err != nil {
		HTTP2ProblemWithError(w, err, http.StatusInternalServerError)

		return
	}

	var hal Hal
	hal = NewBaseHal(nil, NewHalLink(h, nil))

	for factType := range factTypesByHint {
		if h, err := hd.combineURL(HandlerPathOperationBuildFactTemplate, "fact", factType); err != nil {
			HTTP2ProblemWithError(w, err, http.StatusInternalServerError)
		} else {
			hal = hal.AddLink(
				fmt.Sprintf("operation-fact:{%s}", factType),
				NewHalLink(h, nil).SetTemplated(),
			)
		}
	}

	HTTP2WriteHal(hd.enc, w, hal, http.StatusOK)
	HTTP2WriteCache(w, CacheKeyPath(r), time.Hour*100*100*100)
}

func (hd *Handlers) handleOperationBuildFactTemplate(w http.ResponseWriter, r *http.Request) {
	if err := LoadFromCache(hd.cache, CacheKeyPath(r), w); err == nil {
		return
	}

	factType := mux.Vars(r)["fact"]
	hinter, found := factTypesByHint[factType]
	if !found {
		HTTP2ProblemWithError(w, errors.Errorf("unknown operation, %q", factType), http.StatusNotFound)

		return
	}

	builder := NewBuilder(hd.enc, hd.networkID)
	hal, err := builder.FactTemplate(hinter.Hint())
	if err != nil {
		HTTP2ProblemWithError(w, err, http.StatusBadRequest)

		return
	}

	h, err := hd.combineURL(HandlerPathOperationBuildFactTemplate, "fact", factType)
	if err != nil {
		HTTP2ProblemWithError(w, err, http.StatusInternalServerError)

		return
	}
	hal = hal.SetSelf(NewHalLink(h, nil))

	HTTP2WriteHal(hd.enc, w, hal, http.StatusOK)
	HTTP2WriteCache(w, CacheKeyPath(r), time.Hour*100*100*100)
}

func (hd *Handlers) handleOperationBuildFact(w http.ResponseWriter, r *http.Request) {
	body := &bytes.Buffer{}
	if _, err := io.Copy(body, r.Body); err != nil {
		HTTP2ProblemWithError(w, err, http.StatusInternalServerError)

		return
	}

	builder := NewBuilder(hd.enc, hd.networkID)
	hal, err := builder.BuildFact(body.Bytes())
	if err != nil {
		HTTP2ProblemWithError(w, err, http.StatusBadRequest)

		return
	}

	h, err := hd.combineURL(HandlerPathOperationBuildFact)
	if err != nil {
		HTTP2ProblemWithError(w, err, http.StatusInternalServerError)

		return
	}
	hal = hal.SetSelf(NewHalLink(h, nil))

	HTTP2WriteHal(hd.enc, w, hal, http.StatusOK)
}

func (hd *Handlers) handleOperationBuildSign(w http.ResponseWriter, r *http.Request) {
	body := &bytes.Buffer{}
	if _, err := io.Copy(body, r.Body); err != nil {
		HTTP2ProblemWithError(w, err, http.StatusInternalServerError)

		return
	}

	builder := NewBuilder(hd.enc, hd.networkID)
	hal, err := builder.BuildOperation(body.Bytes())
	if err != nil {
		HTTP2ProblemWithError(w, err, http.StatusBadRequest)

		return
	}

	if parseBoolQuery(r.URL.Query().Get("send")) {
		sl, e := hd.sendSeal(hal.Interface())
		if e != nil {
			HTTP2ProblemWithError(w, e, http.StatusInternalServerError)

			return
		}
		hal = hal.SetInterface(sl).AddExtras("sent", true)
	}

	h, err := hd.combineURL(HandlerPathOperationBuildSign)
	if err != nil {
		HTTP2ProblemWithError(w, err, http.StatusInternalServerError)

		return
	}
	hal = hal.SetSelf(NewHalLink(h, nil))

	HTTP2WriteHal(hd.enc, w, hal, http.StatusOK)
}
