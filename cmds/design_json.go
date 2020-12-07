package cmds

import (
	"github.com/spikeekips/mitum/launch/config"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func (no FeeDesign) MarshalJSON() ([]byte, error) {
	m := no.extras
	if m == nil {
		m = map[string]interface{}{}
	}

	m["type"] = no.Type
	m["receiver"] = no.Receiver

	return jsonenc.Marshal(m)
}

type DigestDesignPackerJSON struct {
	Network   config.LocalNetwork `json:"network"`
	Cache     string              `json:"cache"`
	RateLimit *RateLimiterDesign  `json:"rate-limit"`
}

func (no DigestDesign) MarshalJSON() ([]byte, error) {
	var cache string
	if no.cache != nil {
		cache = no.cache.String()
	}
	return jsonenc.Marshal(DigestDesignPackerJSON{
		Network:   no.network,
		Cache:     cache,
		RateLimit: no.RateLimiterYAML,
	})
}

func (no RateLimiterDesign) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(map[string]interface{}{
		"period": *no.PeriodYAML,
		"limit":  *no.Limit,
	})
}
