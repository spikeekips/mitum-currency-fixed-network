package cmds

import (
	"github.com/spikeekips/mitum/launch/config"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func (no FeeerDesign) MarshalJSON() ([]byte, error) {
	m := no.Extras
	if m == nil {
		m = map[string]interface{}{}
	}

	m["type"] = no.Type

	return jsonenc.Marshal(m)
}

type DigestDesignPackerJSON struct {
	Network config.LocalNetwork `json:"network"`
	Cache   string              `json:"cache"`
}

func (no DigestDesign) MarshalJSON() ([]byte, error) {
	var cache string
	if no.cache != nil {
		cache = no.cache.String()
	}
	return jsonenc.Marshal(DigestDesignPackerJSON{
		Network: no.network,
		Cache:   cache,
	})
}
