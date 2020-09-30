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
		hd.notSupported(w)

		return
	}

	if err := loadFromCache(hd.cache, cacheKeyPath(r), w); err != nil {
		hd.Log().Verbose().Err(err).Msg("failed to load cache")
	} else {
		hd.Log().Verbose().Msg("loaded from cache")

		return
	}

	if n, err := hd.nodeInfoHandler(); err != nil {
		hd.Log().Error().Err(err).Msg("failed to get node info")

		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	} else {
		if hal, err := hd.buildNodeInfoHal(n); err != nil {
			hd.problemWithError(w, err, http.StatusInternalServerError)

			return
		} else {
			hd.writeHal(w, hal, http.StatusOK)
			hd.writeCache(w, cacheKeyPath(r), time.Second*2)
		}
	}
}

func (hd *Handlers) buildNodeInfoHal(ni network.NodeInfo) (Hal, error) {
	var hal Hal = NewBaseHal(ni, NewHalLink(HandlerPathNodeInfo, nil))

	if blk := ni.LastBlock(); blk != nil {
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
	}

	return hal, nil
}
