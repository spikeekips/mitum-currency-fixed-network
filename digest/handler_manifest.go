package digest

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (hd *Handlers) handleManifestByHeight(w http.ResponseWriter, r *http.Request) {
	if err := LoadFromCache(hd.cache, CacheKeyPath(r), w); err == nil {
		return
	}

	var height base.Height
	switch h, err := parseHeightFromPath(mux.Vars(r)["height"]); {
	case err != nil:
		HTTP2ProblemWithError(w, errors.Errorf("invalid height found for manifest by height"), http.StatusBadRequest)

		return
	case h <= base.NilHeight:
		HTTP2ProblemWithError(w, errors.Errorf("invalid height, %v", h), http.StatusBadRequest)
		return
	default:
		height = h
	}

	hd.handleManifest(w, r, func() (block.Manifest, bool, error) {
		return hd.database.ManifestByHeight(height)
	})
}

func (hd *Handlers) handleManifestByHash(w http.ResponseWriter, r *http.Request) {
	if err := LoadFromCache(hd.cache, CacheKeyPath(r), w); err == nil {
		return
	}

	var h valuehash.Hash
	h, err := parseHashFromPath(mux.Vars(r)["hash"])
	if err != nil {
		HTTP2ProblemWithError(w, errors.Wrap(err, "invalid hash for manifest by hash"), http.StatusBadRequest)

		return
	}

	hd.handleManifest(w, r, func() (block.Manifest, bool, error) {
		return hd.database.Manifest(h)
	})
}

func (hd *Handlers) handleManifest(w http.ResponseWriter, r *http.Request, get func() (block.Manifest, bool, error)) {
	cachekey := CacheKeyPath(r)
	if v, err, shared := hd.rg.Do(cachekey, func() (interface{}, error) {
		return hd.handleManifestInGroup(get)
	}); err != nil {
		HTTP2HandleError(w, err)
	} else {
		HTTP2WriteHalBytes(hd.enc, w, v.([]byte), http.StatusOK)

		if !shared {
			HTTP2WriteCache(w, cachekey, time.Hour*30)
		}
	}
}

func (hd *Handlers) handleManifestInGroup(get func() (block.Manifest, bool, error)) ([]byte, error) {
	var manifest block.Manifest
	switch m, found, err := get(); {
	case err != nil:
		return nil, err
	case !found:
		return nil, util.NotFoundError.Errorf("manifest not found")
	default:
		manifest = m
	}

	i, err := hd.buildManifestHal(manifest)
	if err != nil {
		return nil, err
	}
	return hd.enc.Marshal(i)
}

func (hd *Handlers) buildManifestHal(manifest block.Manifest) (Hal, error) {
	height := manifest.Height()

	var hal Hal
	h, err := hd.combineURL(HandlerPathManifestByHeight, "height", height.String())
	if err != nil {
		return nil, err
	}
	hal = NewBaseHal(manifest, NewHalLink(h, nil))

	h, err = hd.combineURL(HandlerPathManifestByHash, "hash", manifest.Hash().String())
	if err != nil {
		return nil, err
	}
	hal = hal.AddLink("alternate", NewHalLink(h, nil))

	h, err = hd.combineURL(HandlerPathManifestByHeight, "height", (height + 1).String())
	if err != nil {
		return nil, err
	}
	hal = hal.AddLink("next", NewHalLink(h, nil))

	h, err = hd.combineURL(HandlerPathBlockByHeight, "height", height.String())
	if err != nil {
		return nil, err
	}
	hal = hal.AddLink("block", NewHalLink(h, nil))

	for k := range halBlockTemplate {
		hal = hal.AddLink(k, halBlockTemplate[k])
	}

	return hal, nil
}

func (hd *Handlers) handleManifests(w http.ResponseWriter, r *http.Request) {
	offset := parseOffsetQuery(r.URL.Query().Get("offset"))
	reverse := parseBoolQuery(r.URL.Query().Get("reverse"))

	cachekey := CacheKey(r.URL.Path, stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))
	if err := LoadFromCache(hd.cache, cachekey, w); err == nil {
		return
	}

	height := base.NilHeight
	if len(offset) > 0 {
		ht, err := base.NewHeightFromString(offset)
		if err != nil {
			HTTP2ProblemWithError(w, err, http.StatusBadRequest)

			return
		}
		height = ht
	}

	if v, err, shared := hd.rg.Do(cachekey, func() (interface{}, error) {
		i, filled, err := hd.handleManifestsInGroup(height, offset, reverse)

		return []interface{}{i, filled}, err
	}); err != nil {
		HTTP2HandleError(w, err)
	} else {
		var b []byte
		var filled bool
		{
			l := v.([]interface{})
			b = l[0].([]byte)
			filled = l[1].(bool)
		}

		HTTP2WriteHalBytes(hd.enc, w, b, http.StatusOK)

		if !shared {
			expire := hd.expireNotFilled
			if len(offset) > 0 && filled {
				expire = time.Hour * 30
			}

			HTTP2WriteCache(w, cachekey, expire)
		}
	}
}

func (hd *Handlers) handleManifestsInGroup(height base.Height, offset string, reverse bool) ([]byte, bool, error) {
	limit := hd.itemsLimiter("manifests")

	var vas []Hal
	if err := hd.database.Manifests(
		true, reverse, height, limit,
		func(height base.Height, _ valuehash.Hash, va block.Manifest) (bool, error) {
			if height <= base.PreGenesisHeight {
				return !reverse, nil
			}

			hal, err := hd.buildManifestHal(va)
			if err != nil {
				return false, err
			}
			vas = append(vas, hal)

			return true, nil
		},
	); err != nil {
		return nil, false, err
	} else if len(vas) < 1 {
		return nil, false, util.NotFoundError.Errorf("manifests not found")
	}

	i, err := hd.buildManifestsHAL(vas, offset, reverse)
	if err != nil {
		return nil, false, err
	}
	b, err := hd.enc.Marshal(i)
	return b, int64(len(vas)) == limit, err
}

func (hd *Handlers) buildManifestsHAL(vas []Hal, offset string, reverse bool) (Hal, error) {
	baseSelf, err := hd.combineURL(HandlerPathManifests)
	if err != nil {
		return nil, err
	}
	self := baseSelf
	if len(offset) > 0 {
		self = addQueryValue(baseSelf, stringOffsetQuery(offset))
	}
	if reverse {
		self = addQueryValue(baseSelf, stringBoolQuery("reverse", reverse))
	}
	var hal Hal
	hal = NewBaseHal(vas, NewHalLink(self, nil))

	var nextoffset string
	if len(vas) > 0 {
		va := vas[len(vas)-1].Interface().(block.Manifest)
		nextoffset = va.Height().String()
	}

	if len(nextoffset) > 0 {
		next := baseSelf
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
