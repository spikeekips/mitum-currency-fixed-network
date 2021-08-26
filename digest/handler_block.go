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
	cachekey := CacheKeyPath(r)
	if err := LoadFromCache(hd.cache, cachekey, w); err == nil {
		return
	}

	if v, err, shared := hd.rg.Do(cachekey, func() (interface{}, error) {
		return hd.handleBlockInGroup(mux.Vars(r))
	}); err != nil {
		HTTP2HandleError(w, err)
	} else {
		HTTP2WriteHalBytes(hd.enc, w, v.([]byte), http.StatusOK)
		if !shared {
			HTTP2WriteCache(w, cachekey, time.Hour*3000)
		}
	}
}

func (hd *Handlers) handleBlockInGroup(vars map[string]string) ([]byte, error) {
	var hal Hal
	if s, found := vars["height"]; found {
		height, err := parseHeightFromPath(s)
		if err != nil {
			return nil, quicnetwork.BadRequestError.Errorf("invalid height found for block by height: %w", err)
		}

		h, err := hd.buildBlockHalByHeight(height)
		if err != nil {
			return nil, err
		}
		hal = h
	} else if s, found := vars["hash"]; found {
		h, err := parseHashFromPath(s)
		if err != nil {
			return nil, quicnetwork.BadRequestError.Errorf("invalid hash for block by hash: %w", err)
		}

		i, err := hd.buildBlockHalByHash(h)
		if err != nil {
			return nil, err
		}
		hal = i
	}

	return hd.enc.Marshal(hal)
}

func (hd *Handlers) buildBlockHalByHeight(height base.Height) (Hal, error) {
	h, err := hd.combineURL(HandlerPathBlockByHeight, "height", height.String())
	if err != nil {
		return nil, err
	}

	var hal Hal
	hal = NewBaseHal(nil, NewHalLink(h, nil))

	h, err = hd.combineURL(HandlerPathBlockByHeight, "height", (height + 1).String())
	if err != nil {
		return nil, err
	}
	hal = hal.AddLink("next", NewHalLink(h, nil))

	if height > base.PreGenesisHeight+1 {
		h, err = hd.combineURL(HandlerPathBlockByHeight, "height", (height - 1).String())
		if err != nil {
			return nil, err
		}
		hal = hal.AddLink("prev", NewHalLink(h, nil))
	}

	h, err = hd.combineURL(HandlerPathBlockByHeight, "height", height.String())
	if err != nil {
		return nil, err
	}
	hal = hal.AddLink("current", NewHalLink(h, nil))

	h, err = hd.combineURL(HandlerPathManifestByHeight, "height", height.String())
	if err != nil {
		return nil, err
	}
	hal = hal.AddLink("current-manifest", NewHalLink(h, nil))

	for k := range halBlockTemplate {
		hal = hal.AddLink(k, halBlockTemplate[k])
	}

	return hal, nil
}

func (hd *Handlers) buildBlockHalByHash(h valuehash.Hash) (Hal, error) {
	i, err := hd.combineURL(HandlerPathBlockByHash, "hash", h.String())
	if err != nil {
		return nil, err
	}

	var hal Hal
	hal = NewBaseHal(nil, NewHalLink(i, nil))

	i, err = hd.combineURL(HandlerPathManifestByHash, "hash", h.String())
	if err != nil {
		return nil, err
	}
	hal = hal.AddLink("manifest", NewHalLink(i, nil))

	for k := range halBlockTemplate {
		hal = hal.AddLink(k, halBlockTemplate[k])
	}

	return hal, nil
}
