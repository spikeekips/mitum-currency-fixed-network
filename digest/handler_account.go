package digest

import (
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (hd *Handlers) handleAccount(w http.ResponseWriter, r *http.Request) {
	cachekey := cacheKeyPath(r)
	if err := loadFromCache(hd.cache, cachekey, w); err == nil {
		return
	}

	var address base.Address
	if a, err := base.DecodeAddressFromString(strings.TrimSpace(mux.Vars(r)["address"]), hd.enc); err != nil {
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
		if errors.Is(err, util.NotFoundError) {
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
		hal, err := hd.buildAccountHal(va)
		if err != nil {
			return nil, err
		}
		return hd.enc.Marshal(hal)
	}
}

func (hd *Handlers) buildAccountHal(va AccountValue) (Hal, error) {
	hinted := va.Account().Address().String()
	h, err := hd.combineURL(HandlerPathAccount, "address", hinted)
	if err != nil {
		return nil, err
	}

	var hal Hal
	hal = NewBaseHal(va, NewHalLink(h, nil))

	h, err = hd.combineURL(HandlerPathAccountOperations, "address", hinted)
	if err != nil {
		return nil, err
	}
	hal = hal.
		AddLink("operations", NewHalLink(h, nil)).
		AddLink("operations:{offset}", NewHalLink(h+"?offset={offset}", nil).SetTemplated()).
		AddLink("operations:{offset,reverse}", NewHalLink(h+"?offset={offset}&reverse=1", nil).SetTemplated())

	h, err = hd.combineURL(HandlerPathBlockByHeight, "height", va.Height().String())
	if err != nil {
		return nil, err
	}
	hal = hal.AddLink("block", NewHalLink(h, nil))

	h, err = hd.combineURL(HandlerPathBlockByHeight, "height", va.Height().String())
	if err != nil {
		return nil, err
	}
	hal = hal.AddLink("block", NewHalLink(h, nil))

	if va.PreviousHeight() > base.PreGenesisHeight {
		h, err = hd.combineURL(HandlerPathBlockByHeight, "height", va.PreviousHeight().String())
		if err != nil {
			return nil, err
		}
		hal = hal.AddLink("previous_block", NewHalLink(h, nil))
	}

	return hal, nil
}

func (hd *Handlers) handleAccountOperations(w http.ResponseWriter, r *http.Request) {
	var address base.Address
	if a, err := base.DecodeAddressFromString(strings.TrimSpace(mux.Vars(r)["address"]), hd.enc); err != nil {
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
	if err := loadFromCache(hd.cache, cachekey, w); err == nil {
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
			expire := time.Second * 3
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
			hal, err := hd.buildOperationHal(va)
			if err != nil {
				return false, err
			}
			vas = append(vas, hal)

			return true, nil
		},
	); err != nil {
		return nil, false, err
	} else if len(vas) < 1 {
		return nil, false, util.NotFoundError.Errorf("operations not found")
	}

	i, err := hd.buildAccountOperationsHal(address, vas, offset, reverse)
	if err != nil {
		return nil, false, err
	}

	b, err := hd.enc.Marshal(i)
	return b, int64(len(vas)) == limit, err
}

func (hd *Handlers) buildAccountOperationsHal(
	address base.Address,
	vas []Hal,
	offset string,
	reverse bool,
) (Hal, error) {
	baseSelf, err := hd.combineURL(HandlerPathAccountOperations, "address", address.String())
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

	h, err := hd.combineURL(HandlerPathAccount, "address", address.String())
	if err != nil {
		return nil, err
	}
	hal = hal.AddLink("account", NewHalLink(h, nil))

	var nextoffset string
	if len(vas) > 0 {
		va := vas[len(vas)-1].Interface().(OperationValue)
		nextoffset = buildOffset(va.Height(), va.Index())
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
