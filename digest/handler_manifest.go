package digest

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

func (hd *Handlers) handleManifestByHeight(w http.ResponseWriter, r *http.Request) {
	if err := loadFromCache(hd.cache, cacheKeyPath(r), w); err != nil {
		hd.Log().Verbose().Err(err).Msg("failed to load cache")
	} else {
		hd.Log().Verbose().Msg("loaded from cache")

		return
	}

	var height base.Height
	if h, err := parseHeightFromPath(mux.Vars(r)["height"]); err != nil {
		hd.problemWithError(w, xerrors.Errorf("invalid height found for manifest by height"), http.StatusBadRequest)

		return
	} else {
		height = h
	}

	hd.handleManifest(w, r, func() (block.Manifest, bool, error) {
		return hd.storage.ManifestByHeight(height)
	})
}

func (hd *Handlers) handleManifestByHash(w http.ResponseWriter, r *http.Request) {
	if err := loadFromCache(hd.cache, cacheKeyPath(r), w); err != nil {
		hd.Log().Verbose().Err(err).Msg("failed to load cache")
	} else {
		hd.Log().Verbose().Msg("loaded from cache")

		return
	}

	var h valuehash.Hash
	if b, err := parseHashFromPath(mux.Vars(r)["hash"]); err != nil {
		hd.problemWithError(w, xerrors.Errorf("invalid hash for manifest by hash: %w", err), http.StatusBadRequest)

		return
	} else {
		h = b
	}

	hd.handleManifest(w, r, func() (block.Manifest, bool, error) {
		return hd.storage.Manifest(h)
	})
}

func (hd *Handlers) handleManifest(w http.ResponseWriter, r *http.Request, get func() (block.Manifest, bool, error)) {
	var manifest block.Manifest
	switch m, found, err := get(); {
	case err != nil:
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	case !found:
		hd.problemWithError(w, xerrors.Errorf("manifest not found"), http.StatusNotFound)

		return
	default:
		manifest = m
	}

	if hal, err := hd.buildManifestHal(manifest); err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	} else {
		hd.writeHal(w, hal, http.StatusOK)
		hd.writeCache(w, cacheKeyPath(r), time.Hour*30)
	}
}

func (hd *Handlers) buildManifestHal(manifest block.Manifest) (Hal, error) {
	height := manifest.Height()

	var hal Hal
	if h, err := hd.combineURL(HandlerPathManifestByHeight, "height", height.String()); err != nil {
		return nil, err
	} else {
		hal = NewBaseHal(manifest, NewHalLink(h, nil))
	}

	if h, err := hd.combineURL(HandlerPathManifestByHash, "hash", manifest.Hash().String()); err != nil {
		return nil, err
	} else {
		hal = hal.AddLink("alternate", NewHalLink(h, nil))
	}

	if h, err := hd.combineURL(HandlerPathManifestByHeight, "height", (height + 1).String()); err != nil {
		return nil, err
	} else {
		hal = hal.AddLink("next", NewHalLink(h, nil))
	}

	if h, err := hd.combineURL(HandlerPathBlockByHeight, "height", height.String()); err != nil {
		return nil, err
	} else {
		hal = hal.AddLink("block", NewHalLink(h, nil))
	}

	for k := range halBlockTemplate {
		hal = hal.AddLink(k, halBlockTemplate[k])
	}

	return hal, nil
}
