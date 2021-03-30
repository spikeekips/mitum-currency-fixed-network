package digest

import (
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

func (hd *Handlers) handleAccount(w http.ResponseWriter, r *http.Request) {
	cachekey := cacheKeyPath(r)
	if err := loadFromCache(hd.cache, cachekey, w); err != nil {
		hd.Log().Verbose().Err(err).Msg("failed to load cache")
	} else {
		hd.Log().Verbose().Msg("loaded from cache")

		return
	}

	var address base.Address
	if a, err := base.DecodeAddressFromString(hd.enc, strings.TrimSpace(mux.Vars(r)["address"])); err != nil {
		hd.problemWithError(w, err, http.StatusBadRequest)

		return
	} else if err := a.IsValid(nil); err != nil {
		hd.problemWithError(w, err, http.StatusBadRequest)
		return
	} else {
		address = a
	}

	if v, err, shared := hd.rg.Do(cachekey, func() (interface{}, error) {
		return hd.handleAccountInGroup(address)
	}); err != nil {
		if xerrors.Is(err, util.NotFoundError) {
			err = util.NotFoundError.Errorf("account, %s not found", address)
		} else {
			hd.Log().Error().Err(err).Str("address", address.String()).Msg("failed to get account")
		}

		hd.handleError(w, err)
	} else {
		hd.writeHalBytes(w, v.([]byte), http.StatusOK)

		if !shared {
			hd.writeCache(w, cachekey, time.Second*2)
		}
	}
}

func (hd *Handlers) handleAccountInGroup(address base.Address) (interface{}, error) {
	switch va, found, err := hd.database.Account(address); {
	case err != nil:
		return nil, err
	case !found:
		return nil, util.NotFoundError
	default:
		if hal, err := hd.buildAccountHal(va); err != nil {
			return nil, err
		} else {
			return hd.enc.Marshal(hal)
		}
	}
}

func (hd *Handlers) buildAccountHal(va AccountValue) (Hal, error) {
	var hal Hal
	hinted := va.Account().Address().String()
	if h, err := hd.combineURL(HandlerPathAccount, "address", hinted); err != nil {
		return nil, err
	} else {
		hal = NewBaseHal(va, NewHalLink(h, nil))
	}

	if h, err := hd.combineURL(HandlerPathAccountOperations, "address", hinted); err != nil {
		return nil, err
	} else {
		hal = hal.
			AddLink("operations", NewHalLink(h, nil)).
			AddLink("operations:{offset}", NewHalLink(h+"?offset={offset}", nil).SetTemplated()).
			AddLink("operations:{offset,reverse}", NewHalLink(h+"?offset={offset}&reverse=1", nil).SetTemplated())
	}

	if h, err := hd.combineURL(HandlerPathBlockByHeight, "height", va.Height().String()); err != nil {
		return nil, err
	} else {
		hal = hal.AddLink("block", NewHalLink(h, nil))
	}

	if h, err := hd.combineURL(HandlerPathBlockByHeight, "height", va.Height().String()); err != nil {
		return nil, err
	} else {
		hal = hal.AddLink("block", NewHalLink(h, nil))
	}

	if va.PreviousHeight() > base.PreGenesisHeight {
		if h, err := hd.combineURL(HandlerPathBlockByHeight, "height", va.PreviousHeight().String()); err != nil {
			return nil, err
		} else {
			hal = hal.AddLink("previous_block", NewHalLink(h, nil))
		}
	}

	return hal, nil
}

func (hd *Handlers) handleAccountOperations(w http.ResponseWriter, r *http.Request) {
	var address base.Address
	if a, err := base.DecodeAddressFromString(hd.enc, strings.TrimSpace(mux.Vars(r)["address"])); err != nil {
		hd.problemWithError(w, err, http.StatusBadRequest)

		return
	} else if err := a.IsValid(nil); err != nil {
		hd.problemWithError(w, err, http.StatusBadRequest)
		return
	} else {
		address = a
	}

	offset := parseOffsetQuery(r.URL.Query().Get("offset"))
	reverse := parseBoolQuery(r.URL.Query().Get("reverse"))

	cachekey := cacheKey(r.URL.Path, stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))
	if err := loadFromCache(hd.cache, cachekey, w); err != nil {
		hd.Log().Verbose().Err(err).Msg("failed to load cache")
	} else {
		hd.Log().Verbose().Msg("loaded from cache")
		return
	}

	if v, err, shared := hd.rg.Do(cachekey, func() (interface{}, error) {
		i, filled, err := hd.handleAccountOperationsInGroup(address, offset, reverse)

		return []interface{}{i, filled}, err
	}); err != nil {
		hd.handleError(w, err)
	} else {
		var b []byte
		var filled bool
		{
			l := v.([]interface{})
			b = l[0].([]byte)
			filled = l[1].(bool)
		}

		hd.writeHalBytes(w, b, http.StatusOK)

		if !shared {
			var expire time.Duration = time.Second * 3
			if filled {
				expire = time.Hour * 30
			}

			hd.writeCache(w, cachekey, expire)
		}
	}
}

func (hd *Handlers) handleAccountOperationsInGroup(
	address base.Address,
	offset string,
	reverse bool,
) ([]byte, bool, error) {
	limit := hd.itemsLimiter("account-operations")
	var vas []Hal
	if err := hd.database.OperationsByAddress(
		address, true, reverse, offset, limit,
		func(_ valuehash.Hash, va OperationValue) (bool, error) {
			if hal, err := hd.buildOperationHal(va); err != nil {
				return false, err
			} else {
				vas = append(vas, hal)
			}

			return true, nil
		},
	); err != nil {
		return nil, false, err
	} else if len(vas) < 1 {
		return nil, false, util.NotFoundError.Errorf("operations not found")
	}

	if i, err := hd.buildAccountOperationsHal(address, vas, offset, reverse); err != nil {
		return nil, false, err
	} else {
		b, err := hd.enc.Marshal(i)
		return b, int64(len(vas)) == limit, err
	}
}

func (hd *Handlers) buildAccountOperationsHal(
	address base.Address,
	vas []Hal,
	offset string,
	reverse bool,
) (Hal, error) {
	var hal Hal
	var baseSelf string
	if h, err := hd.combineURL(HandlerPathAccountOperations, "address", address.String()); err != nil {
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

	if h, err := hd.combineURL(HandlerPathAccount, "address", address.String()); err != nil {
		return nil, err
	} else {
		hal = hal.AddLink("account", NewHalLink(h, nil))
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
