package digest

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

var halBlockTemplate = map[string]HalLink{
	"block:{height}":    NewHalLink(HandlerPathBlockByHeight, nil).SetTemplated(),
	"block:{hash}":      NewHalLink(HandlerPathBlockByHeight, nil).SetTemplated(),
	"manifest:{height}": NewHalLink(HandlerPathManifestByHeight, nil).SetTemplated(),
	"manifest:{hash}":   NewHalLink(HandlerPathManifestByHash, nil).SetTemplated(),
}

func (hd *Handlers) handleBlock(w http.ResponseWriter, r *http.Request) {
	if err := loadFromCache(hd.cache, cacheKeyPath(r), w); err != nil {
		hd.Log().Verbose().Err(err).Msg("failed to load cache")
	} else {
		hd.Log().Verbose().Msg("loaded from cache")

		return
	}

	vars := mux.Vars(r)

	var hal Hal
	if s, found := vars["height"]; found {
		var height base.Height
		if h, err := parseHeightFromPath(s); err != nil {
			hd.problemWithError(w, xerrors.Errorf("invalid height found for block by height: %w", err), http.StatusBadRequest)

			return
		} else {
			height = h
		}

		if h, err := hd.buildBlockHalByHeight(height); err != nil {
			hd.problemWithError(w, err, http.StatusInternalServerError)

			return
		} else {
			hal = h
		}
	} else if s, found := vars["hash"]; found {
		var h valuehash.Hash
		if b, err := parseHashFromPath(s); err != nil {
			hd.problemWithError(w, xerrors.Errorf("invalid hash for block by hash: %w", err), http.StatusBadRequest)

			return
		} else {
			h = b
		}

		if h, err := hd.buildBlockHalByHash(h); err != nil {
			hd.problemWithError(w, err, http.StatusInternalServerError)

			return
		} else {
			hal = h
		}
	}

	hd.writeHal(w, hal, http.StatusOK)
	hd.writeCache(w, cacheKeyPath(r), time.Hour*3000)
}

func (hd *Handlers) buildBlockHalByHeight(height base.Height) (Hal, error) {
	var hal Hal
	if h, err := hd.combineURL(HandlerPathBlockByHeight, "height", height.String()); err != nil {
		return nil, err
	} else {
		hal = NewBaseHal(nil, NewHalLink(h, nil))
	}

	if h, err := hd.combineURL(HandlerPathBlockByHeight, "height", (height + 1).String()); err != nil {
		return nil, err
	} else {
		hal = hal.AddLink("next", NewHalLink(h, nil))
	}

	if height > base.PreGenesisHeight+1 {
		if h, err := hd.combineURL(HandlerPathBlockByHeight, "height", (height - 1).String()); err != nil {
			return nil, err
		} else {
			hal = hal.AddLink("prev", NewHalLink(h, nil))
		}
	}

	if h, err := hd.combineURL(HandlerPathBlockByHeight, "height", height.String()); err != nil {
		return nil, err
	} else {
		hal = hal.AddLink("latest", NewHalLink(h, nil))
	}

	if h, err := hd.combineURL(HandlerPathManifestByHeight, "height", height.String()); err != nil {
		return nil, err
	} else {
		hal = hal.AddLink("latest-manifest", NewHalLink(h, nil))
	}

	for k := range halBlockTemplate {
		hal = hal.AddLink(k, halBlockTemplate[k])
	}

	return hal, nil
}

func (hd *Handlers) buildBlockHalByHash(h valuehash.Hash) (Hal, error) {
	var hal Hal
	if h, err := hd.combineURL(HandlerPathBlockByHash, "hash", h.String()); err != nil {
		return nil, err
	} else {
		hal = NewBaseHal(nil, NewHalLink(h, nil))
	}

	if h, err := hd.combineURL(HandlerPathManifestByHash, "hash", h.String()); err != nil {
		return nil, err
	} else {
		hal = hal.AddLink("manifest", NewHalLink(h, nil))
	}

	for k := range halBlockTemplate {
		hal = hal.AddLink(k, halBlockTemplate[k])
	}

	return hal, nil
}
