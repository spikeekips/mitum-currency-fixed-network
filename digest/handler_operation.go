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
			hal = hal.AddLink("block:{height}", NewHalLink(HandlerPathBlockByHeight, nil).SetTemplated())

			hd.writeHal(w, hal, http.StatusOK)
			hd.writeCache(w, cacheKeyPath(r), time.Hour*30)
		}
	}
}

func (hd *Handlers) handleOperations(w http.ResponseWriter, r *http.Request) {
	offset := parseOffsetQuery(r.URL.Query().Get("offset"))
	reverse := parseBoolQuery(r.URL.Query().Get("reverse"))

	ckey := cacheKey(r.URL.Path, stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))
	if err := loadFromCache(hd.cache, ckey, w); err != nil {
		hd.Log().Verbose().Err(err).Msg("failed to load cache")
	} else {
		hd.Log().Verbose().Msg("loaded from cache")

		return
	}

	var vas []Hal
	if err := hd.storage.Operations(
		true, reverse, offset, hd.itemsLimiter("operations"),
		func(_ valuehash.Hash, va OperationValue) (bool, error) {
			if hal, err := hd.buildOperationHal(va); err != nil {
				return false, err
			} else {
				vas = append(vas, hal)
			}

			return true, nil
		},
	); err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	} else if len(vas) < 1 {
		hd.problemWithError(w, xerrors.Errorf("operations not found"), http.StatusNotFound)

		return
	}

	if hal, err := hd.buildOperationsHal(vas, offset, reverse); err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	} else {
		hd.writeHal(w, hal, http.StatusOK)
		hd.writeCache(w, ckey, time.Second*2) // TODO too short expire time.
	}
}

func (hd *Handlers) buildOperationHal(va OperationValue) (Hal, error) {
	var hal Hal

	if h, err := hd.combineURL(HandlerPathOperation, "hash", va.Operation().Fact().Hash().String()); err != nil {
		return nil, err
	} else {
		hal = NewBaseHal(va, NewHalLink(h, nil))
	}

	if h, err := hd.combineURL(HandlerPathBlockByHeight, "height", va.Height().String()); err != nil {
		return nil, err
	} else {
		hal = hal.AddLink("block", NewHalLink(h, nil))
	}

	if h, err := hd.combineURL(HandlerPathManifestByHeight, "height", va.Height().String()); err != nil {
		return nil, err
	} else {
		hal = hal.AddLink("manifest", NewHalLink(h, nil))
	}

	return hal, nil
}

func (hd *Handlers) buildOperationsHal(
	vas []Hal,
	offset string,
	reverse bool,
) (Hal, error) {
	var hal Hal
	var baseSelf string
	if h, err := hd.combineURL(HandlerPathOperations); err != nil {
		return nil, err
	} else {
		baseSelf = h

		var self string = baseSelf
		if len(offset) > 0 {
			self = addQueryValue(baseSelf, stringOffsetQuery(offset))
		}
		if reverse {
			self = addQueryValue(h, stringBoolQuery("reverse", reverse))
		}
		hal = NewBaseHal(vas, NewHalLink(self, nil))
	}

	var nextoffset string
	if len(vas) > 0 {
		va := vas[len(vas)-1].Interface().(OperationValue)
		nextoffset = buildOffset(va.Height(), va.Index())
	}

	if len(nextoffset) > 0 {
		var next string = baseSelf
		if len(nextoffset) > 0 {
			next = addQueryValue(next, stringOffsetQuery(nextoffset))
		}

		if reverse {
			next = addQueryValue(next, stringBoolQuery("reverse", reverse))
		}

		hal = hal.AddLink("next", NewHalLink(next, nil))
	}

	hal = hal.AddLink("reverse", NewHalLink(addQueryValue(baseSelf, stringBoolQuery("reverse", !reverse)), nil))

	return hal, nil
}
