package digest

import (
	"net/http"
	"strings"
	"time"

	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/network"
)

func (hd *Handlers) SetNodeInfoHandler(handler network.NodeInfoHandler) *Handlers {
	hd.nodeInfoHandler = handler

	return hd
}

func (hd *Handlers) handleNodeInfo(w http.ResponseWriter, r *http.Request) {
	if hd.nodeInfoHandler == nil {
		hd.notSupported(w, nil)

		return
	}

	cachekey := cacheKeyPath(r)
	if err := loadFromCache(hd.cache, cachekey, w); err != nil {
		hd.Log().Verbose().Err(err).Msg("failed to load cache")
	} else {
		hd.Log().Verbose().Msg("loaded from cache")

		return
	}

	if v, err, shared := hd.rg.Do(cachekey, hd.handleNodeInfoInGroup); err != nil {
		hd.Log().Error().Err(err).Msg("failed to get node info")

		hd.handleError(w, err)
	} else {
		hd.writeHalBytes(w, v.([]byte), http.StatusOK)

		if !shared {
			hd.writeCache(w, cachekey, time.Second*3)
		}
	}
}

func (hd *Handlers) handleNodeInfoInGroup() (interface{}, error) {
	if n, err := hd.nodeInfoHandler(); err != nil {
		return nil, err
	} else if i, err := hd.buildNodeInfoHal(n); err != nil {
		return nil, err
	} else {
		return hd.enc.Marshal(i)
	}
}

func (hd *Handlers) buildNodeInfoHal(ni network.NodeInfo) (Hal, error) {
	var hal Hal = NewBaseHal(ni, NewHalLink(HandlerPathNodeInfo, nil))

	hal = hal.AddLink("currency", NewHalLink(HandlerPathCurrencies, nil)).
		AddLink("currency:{currencyid}", NewHalLink(HandlerPathCurrency, nil).SetTemplated())

	var blk block.Manifest
	if i := ni.LastBlock(); i == nil {
		return hal, nil
	} else {
		blk = i
	}

	if bh, err := hd.buildBlockHalByHeight(blk.Height()); err != nil {
		return nil, err
	} else {
		for k, v := range bh.Links() {
			if !strings.HasPrefix(k, "block:") {
				k = "block:" + k
			}
			hal = hal.AddLink(k, v)
		}
	}

	return hal, nil
}
