package digest

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/spikeekips/mitum/base"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/util/valuehash"
)

var halBlockTemplate = map[string]HalLink{
	"block:{height}":    NewHalLink(HandlerPathBlockByHeight, nil).SetTemplated(),
	"block:{hash}":      NewHalLink(HandlerPathBlockByHeight, nil).SetTemplated(),
	"manifest:{height}": NewHalLink(HandlerPathManifestByHeight, nil).SetTemplated(),
	"manifest:{hash}":   NewHalLink(HandlerPathManifestByHash, nil).SetTemplated(),
}

func (hd *Handlers) handleBlock(w http.ResponseWriter, r *http.Request) {
	cachekey := cacheKeyPath(r)
	if err := loadFromCache(hd.cache, cachekey, w); err != nil {
		hd.Log().Verbose().Err(err).Msg("failed to load cache")
	} else {
		hd.Log().Verbose().Msg("loaded from cache")

		return
	}

	if v, err, shared := hd.rg.Do(cachekey, func() (interface{}, error) {
		return hd.handleBlockInGroup(mux.Vars(r))
	}); err != nil {
		hd.handleError(w, err)
	} else {
		hd.writeHalBytes(w, v.([]byte), http.StatusOK)
		if !shared {
			hd.writeCache(w, cachekey, time.Hour*3000)
		}
	}
}

func (hd *Handlers) handleBlockInGroup(vars map[string]string) ([]byte, error) {
	var hal Hal
	if s, found := vars["height"]; found {
		var height base.Height
		if h, err := parseHeightFromPath(s); err != nil {
			return nil, quicnetwork.BadRequestError.Errorf("invalid height found for block by height: %w", err)
		} else {
			height = h
		}

		if h, err := hd.buildBlockHalByHeight(height); err != nil {
			return nil, err
		} else {
			hal = h
		}
	} else if s, found := vars["hash"]; found {
		var h valuehash.Hash
		if b, err := parseHashFromPath(s); err != nil {
			return nil, quicnetwork.BadRequestError.Errorf("invalid hash for block by hash: %w", err)
		} else {
			h = b
		}

		if h, err := hd.buildBlockHalByHash(h); err != nil {
			return nil, err
		} else {
			hal = h
		}
	}

	return hd.enc.Marshal(hal)
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
		hal = hal.AddLink("current", NewHalLink(h, nil))
	}

	if h, err := hd.combineURL(HandlerPathManifestByHeight, "height", height.String()); err != nil {
		return nil, err
	} else {
		hal = hal.AddLink("current-manifest", NewHalLink(h, nil))
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
