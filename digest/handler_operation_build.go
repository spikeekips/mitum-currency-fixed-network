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
	"create-accounts":   currency.CreateAccounts{},
	"key-updater":       currency.KeyUpdater{},
	"transfers":         currency.Transfers{},
	"currency-register": currency.CurrencyRegister{},
}

func (hd *Handlers) handleOperationBuild(w http.ResponseWriter, r *http.Request) {
	if err := loadFromCache(hd.cache, cacheKeyPath(r), w); err == nil {
		return
	}

	h, err := hd.combineURL(HandlerPathOperationBuild)
	if err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	}

	var hal Hal
	hal = NewBaseHal(nil, NewHalLink(h, nil))

	for factType := range factTypesByHint {
		if h, err := hd.combineURL(HandlerPathOperationBuildFactTemplate, "fact", factType); err != nil {
			hd.problemWithError(w, err, http.StatusInternalServerError)
		} else {
			hal = hal.AddLink(
				fmt.Sprintf("operation-fact:{%s}", factType),
				NewHalLink(h, nil).SetTemplated(),
			)
		}
	}

	hd.writeHal(w, hal, http.StatusOK)
	hd.writeCache(w, cacheKeyPath(r), time.Hour*100*100*100)
}

func (hd *Handlers) handleOperationBuildFactTemplate(w http.ResponseWriter, r *http.Request) {
	if err := loadFromCache(hd.cache, cacheKeyPath(r), w); err == nil {
		return
	}

	factType := mux.Vars(r)["fact"]
	hinter, found := factTypesByHint[factType]
	if !found {
		hd.problemWithError(w, errors.Errorf("unknown operation, %q", factType), http.StatusNotFound)

		return
	}

	builder := NewBuilder(hd.enc, hd.networkID)
	hal, err := builder.FactTemplate(hinter.Hint())
	if err != nil {
		hd.problemWithError(w, err, http.StatusBadRequest)

		return
	}

	h, err := hd.combineURL(HandlerPathOperationBuildFactTemplate, "fact", factType)
	if err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	}
	hal = hal.SetSelf(NewHalLink(h, nil))

	hd.writeHal(w, hal, http.StatusOK)
	hd.writeCache(w, cacheKeyPath(r), time.Hour*100*100*100)
}

func (hd *Handlers) handleOperationBuildFact(w http.ResponseWriter, r *http.Request) {
	body := &bytes.Buffer{}
	if _, err := io.Copy(body, r.Body); err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	}

	builder := NewBuilder(hd.enc, hd.networkID)
	hal, err := builder.BuildFact(body.Bytes())
	if err != nil {
		hd.problemWithError(w, err, http.StatusBadRequest)

		return
	}

	h, err := hd.combineURL(HandlerPathOperationBuildFact)
	if err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	}
	hal = hal.SetSelf(NewHalLink(h, nil))

	hd.writeHal(w, hal, http.StatusOK)
}

func (hd *Handlers) handleOperationBuildSign(w http.ResponseWriter, r *http.Request) {
	body := &bytes.Buffer{}
	if _, err := io.Copy(body, r.Body); err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	}

	builder := NewBuilder(hd.enc, hd.networkID)
	hal, err := builder.BuildOperation(body.Bytes())
	if err != nil {
		hd.problemWithError(w, err, http.StatusBadRequest)

		return
	}

	if parseBoolQuery(r.URL.Query().Get("send")) {
		sl, e := hd.sendSeal(hal.Interface())
		if e != nil {
			hd.problemWithError(w, e, http.StatusInternalServerError)

			return
		}
		hal = hal.SetInterface(sl).AddExtras("sent", true)
	}

	h, err := hd.combineURL(HandlerPathOperationBuildSign)
	if err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	}
	hal = hal.SetSelf(NewHalLink(h, nil))

	hd.writeHal(w, hal, http.StatusOK)
}
