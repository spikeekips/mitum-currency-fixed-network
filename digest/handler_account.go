package digest

import (
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

func (hd *Handlers) handleAccount(w http.ResponseWriter, r *http.Request) {
	if err := loadFromCache(hd.cache, cacheKeyPath(r), w); err != nil {
		hd.Log().Verbose().Err(err).Msg("failed to load cache")
	} else {
		hd.Log().Verbose().Msg("loaded from cache")

		return
	}

	var address base.Address
	if a, err := base.DecodeAddressFromString(hd.enc, strings.TrimSpace(mux.Vars(r)["address"])); err != nil {
		hd.problemWithError(w, err, http.StatusBadRequest)

		return
	} else {
		address = a
	}

	switch va, found, err := hd.storage.Account(address); {
	case err != nil:
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	case !found:
		hd.problemWithError(w, xerrors.Errorf("account not found"), http.StatusNotFound)

		return
	default:
		if hal, err := hd.buildAccountHal(va); err != nil {
			hd.problemWithError(w, err, http.StatusNotFound)

			return
		} else {
			hd.writeHal(w, hal, http.StatusOK)
			hd.writeCache(w, cacheKeyPath(r), time.Hour*30)
		}
	}
}

func (hd *Handlers) buildAccountHal(va AccountValue) (Hal, error) {
	var hal Hal
	hinted := currency.AddressToHintedString(va.Account().Address())
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
	} else {
		address = a
	}

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
	if err := hd.storage.OperationsByAddress(
		address, true, reverse, offset, hd.itemsLimiter("account-operations"),
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

	if hal, err := hd.buildAccountOperationsHal(address, vas, offset, reverse); err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	} else {
		hd.writeHal(w, hal, http.StatusOK)
		hd.writeCache(w, ckey, time.Hour*30)
	}
}

func (hd *Handlers) buildAccountOperationsHal(
	address base.Address,
	vas []Hal,
	offset string,
	reverse bool,
) (Hal, error) {
	var hal Hal
	hinted := currency.AddressToHintedString(address)

	var baseSelf string
	if h, err := hd.combineURL(HandlerPathAccountOperations, "address", hinted); err != nil {
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

	if h, err := hd.combineURL(HandlerPathAccount, "address", hinted); err != nil {
		return nil, err
	} else {
		hal = hal.AddLink("account", NewHalLink(h, nil))
	}

	var nextoffset string
	if len(vas) > 0 {
		va := vas[len(vas)-1].Interface().(OperationValue)
		nextoffset = buildOffset(va.Height(), va.Operation().Fact().Hash().String())
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

	return hal, nil
}
