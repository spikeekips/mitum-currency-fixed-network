package digest

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

func (hd *Handlers) handleOperation(w http.ResponseWriter, r *http.Request) {
	if err := loadFromCache(hd.cache, cacheKeyPath(r), w); err != nil {
		hd.Log().Verbose().Err(err).Msg("failed to load cache")
	} else {
		hd.Log().Verbose().Msg("loaded from cache")

		return
	}

	var h valuehash.Hash
	if b, err := parseHashFromPath(mux.Vars(r)["hash"]); err != nil {
		hd.problemWithError(w, xerrors.Errorf("invalid hash for operation by hash: %w", err), http.StatusBadRequest)

		return
	} else {
		h = b
	}

	switch va, found, err := hd.storage.Operation(h, true); {
	case err != nil:
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	case !found:
		hd.problemWithError(w, xerrors.Errorf("operation not found"), http.StatusNotFound)

		return
	default:
		if hal, err := hd.buildOperationHal(va); err != nil {
			hd.problemWithError(w, err, http.StatusInternalServerError)

			return
		} else {
			hal = hal.AddLink("operation:{hash}", NewHalLink(HandlerPathOperation, nil).SetTemplated())
			hal = hal.AddLink("account:{address}", NewHalLink(HandlerPathAccount, nil).SetTemplated())
			hal = hal.AddLink("block:{height}", NewHalLink(HandlerPathBlockByHeight, nil).SetTemplated())

			hd.writeHal(w, hal, http.StatusOK)
			hd.writeCache(w, cacheKeyPath(r), time.Hour*30)
		}
	}
}
