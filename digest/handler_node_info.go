package digest

import (
	"net/http"
	"strings"
	"time"

	"github.com/spikeekips/mitum/network"
)

func (hd *Handlers) SetNodeInfoHandler(handler network.NodeInfoHandler) *Handlers {
	hd.nodeInfoHandler = handler

	return hd
}

func (hd *Handlers) handleNodeInfo(w http.ResponseWriter, r *http.Request) {
	if hd.nodeInfoHandler == nil {
		HTTP2NotSupported(w, nil)

		return
	}

	cachekey := CacheKeyPath(r)
	if err := LoadFromCache(hd.cache, cachekey, w); err == nil {
		return
	}

	if v, err, shared := hd.rg.Do(cachekey, hd.handleNodeInfoInGroup); err != nil {
		hd.Log().Error().Err(err).Msg("failed to get node info")

		HTTP2HandleError(w, err)
	} else {
		HTTP2WriteHalBytes(hd.enc, w, v.([]byte), http.StatusOK)

		if !shared {
			HTTP2WriteCache(w, cachekey, time.Second*3)
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

	blk := ni.LastBlock()
	if blk == nil {
		return hal, nil
	}

	bh, err := hd.buildBlockHalByHeight(blk.Height())
	if err != nil {
		return nil, err
	}
	for k, v := range bh.Links() {
		if !strings.HasPrefix(k, "block:") {
			k = "block:" + k
		}
		hal = hal.AddLink(k, v)
	}

	return hal, nil
}
