package digest

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/util/hint"
	"golang.org/x/xerrors"
)

var factTypesByHint = map[string]hint.Hinter{
	"create-accounts": currency.CreateAccounts{},
	"key-updater":     currency.KeyUpdater{},
	"transfers":       currency.Transfers{},
}

func (hd *Handlers) handleOperationBuild(w http.ResponseWriter, r *http.Request) {
	if err := loadFromCache(hd.cache, cacheKeyPath(r), w); err != nil {
		hd.Log().Verbose().Err(err).Msg("failed to load cache")
	} else {
		hd.Log().Verbose().Msg("loaded from cache")

		return
	}

	var hal Hal
	if h, err := hd.combineURL(HandlerPathOperationBuild); err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	} else {
		hal = NewBaseHal(nil, NewHalLink(h, nil))
	}

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
	if err := loadFromCache(hd.cache, cacheKeyPath(r), w); err != nil {
		hd.Log().Verbose().Err(err).Msg("failed to load cache")
	} else {
		hd.Log().Verbose().Msg("loaded from cache")

		return
	}

	factType := mux.Vars(r)["fact"]
	var hinter hint.Hinter
	if ht, found := factTypesByHint[factType]; !found {
		hd.problemWithError(w, xerrors.Errorf("unknown operation, %q", factType), http.StatusInternalServerError)

		return
	} else {
		hinter = ht
	}

	var hal Hal

	builder := NewBuilder(hd.enc, hd.networkID)
	if h, err := builder.FactTemplate(hinter.Hint()); err != nil {
		hd.problemWithError(w, err, http.StatusBadRequest)

		return
	} else {
		hal = h
	}

	if h, err := hd.combineURL(HandlerPathOperationBuildFactTemplate, "fact", factType); err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	} else {
		hal = hal.SetSelf(NewHalLink(h, nil))
	}

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
	var hal Hal
	if h, err := builder.BuildFact(body.Bytes()); err != nil {
		hd.problemWithError(w, err, http.StatusBadRequest)

		return
	} else {
		hal = h
	}

	if h, err := hd.combineURL(HandlerPathOperationBuildFact); err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	} else {
		hal = hal.SetSelf(NewHalLink(h, nil))
	}

	hd.writeHal(w, hal, http.StatusOK)
}

func (hd *Handlers) handleOperationBuildSign(w http.ResponseWriter, r *http.Request) {
	body := &bytes.Buffer{}
	if _, err := io.Copy(body, r.Body); err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	}

	var hal Hal
	builder := NewBuilder(hd.enc, hd.networkID)
	if h, err := builder.BuildOperation(body.Bytes()); err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	} else {
		hal = h
	}

	if parseBoolQuery(r.URL.Query().Get("send")) {
		if sl, err := hd.sendSeal(hal.Interface()); err != nil {
			hd.problemWithError(w, err, http.StatusInternalServerError)

			return
		} else {
			hal = hal.SetInterface(sl).AddExtras("sent", true)
		}
	}

	if h, err := hd.combineURL(HandlerPathOperationBuildSign); err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	} else {
		hal = hal.SetSelf(NewHalLink(h, nil))
	}

	hd.writeHal(w, hal, http.StatusOK)
}
