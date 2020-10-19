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

func (hd *Handlers) handleManifests(w http.ResponseWriter, r *http.Request) {
	offset := parseOffsetQuery(r.URL.Query().Get("offset"))
	reverse := parseBoolQuery(r.URL.Query().Get("reverse"))

	ckey := cacheKey(r.URL.Path, stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))
	if err := loadFromCache(hd.cache, ckey, w); err != nil {
		hd.Log().Verbose().Err(err).Msg("failed to load cache")
	} else {
		hd.Log().Verbose().Msg("loaded from cache")

		return
	}

	var height base.Height = base.NilHeight
	if len(offset) > 0 {
		if ht, err := base.NewHeightFromString(offset); err != nil {
			hd.problemWithError(w, err, http.StatusBadRequest)

			return
		} else {
			height = ht
		}
	}

	var vas []Hal
	if err := hd.storage.Manifests(
		true, reverse, height, hd.itemsLimiter("manifests"),
		func(height base.Height, _ valuehash.Hash, va block.Manifest) (bool, error) {
			if height <= base.PreGenesisHeight {
				return !reverse, nil
			}

			if hal, err := hd.buildManifestHal(va); err != nil {
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
		hd.problemWithError(w, xerrors.Errorf("manifests not found"), http.StatusNotFound)

		return
	}

	if hal, err := hd.buildManifestsHAL(vas, offset, reverse); err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	} else {
		hd.writeHal(w, hal, http.StatusOK)
		hd.writeCache(w, ckey, time.Second*2) // TODO too short expire time.
	}
}

func (hd *Handlers) buildManifestsHAL(vas []Hal, offset string, reverse bool) (Hal, error) {
	var hal Hal
	var baseSelf string
	if h, err := hd.combineURL(HandlerPathManifests); err != nil {
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
		va := vas[len(vas)-1].Interface().(block.Manifest)
		nextoffset = va.Height().String()
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
