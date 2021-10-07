package digest

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/ulule/limiter/v3"
	"golang.org/x/sync/singleflight"
)

var (
	HTTP2EncoderHintHeader = http.CanonicalHeaderKey("x-mitum-encoder-hint")
	HALMimetype            = "application/hal+json; charset=utf-8"
)

var (
	HandlerPathNodeInfo                   = `/`
	HandlerPathCurrencies                 = `/currency`
	HandlerPathCurrency                   = `/currency/{currencyid:.*}`
	HandlerPathManifests                  = `/block/manifests`
	HandlerPathOperations                 = `/block/operations`
	HandlerPathOperation                  = `/block/operation/{hash:(?i)[0-9a-z][0-9a-z]+}`
	HandlerPathBlockByHeight              = `/block/{height:[0-9]+}`
	HandlerPathBlockByHash                = `/block/{hash:(?i)[0-9a-z][0-9a-z]+}`
	HandlerPathOperationsByHeight         = `/block/{height:[0-9]+}/operations`
	HandlerPathManifestByHeight           = `/block/{height:[0-9]+}/manifest`
	HandlerPathManifestByHash             = `/block/{hash:(?i)[0-9a-z][0-9a-z]+}/manifest`
	HandlerPathAccount                    = `/account/{address:(?i)[0-9a-z][0-9a-z\-]+:[a-z0-9][a-z0-9\-_\+]*[a-z0-9]-v[0-9\.]*}`            // revive:disable-line:line-length-limit
	HandlerPathAccountOperations          = `/account/{address:(?i)[0-9a-z][0-9a-z\-]+:[a-z0-9][a-z0-9\-_\+]*[a-z0-9]-v[0-9\.]*}/operations` // revive:disable-line:line-length-limit
	HandlerPathAccounts                   = `/accounts`
	HandlerPathOperationBuildFactTemplate = `/builder/operation/fact/template/{fact:[\w][\w\-]*}`
	HandlerPathOperationBuildFact         = `/builder/operation/fact`
	HandlerPathOperationBuildSign         = `/builder/operation/sign`
	HandlerPathOperationBuild             = `/builder/operation`
	HandlerPathSend                       = `/builder/send`
)

var RateLimitHandlerMap = map[string]string{
	"node-info":                       HandlerPathNodeInfo,
	"currencies":                      HandlerPathCurrencies,
	"currency":                        HandlerPathCurrency,
	"block-manifests":                 HandlerPathManifests,
	"block-operations":                HandlerPathOperations,
	"block-operation":                 HandlerPathOperation,
	"block-by-height":                 HandlerPathBlockByHeight,
	"block-by-hash":                   HandlerPathBlockByHash,
	"block-operations-by-height":      HandlerPathOperationsByHeight,
	"block-manifest-by-height":        HandlerPathManifestByHeight,
	"block-manifest-by-hash":          HandlerPathManifestByHash,
	"account":                         HandlerPathAccount,
	"account-operations":              HandlerPathAccountOperations,
	"accounts":                        HandlerPathAccounts,
	"builder-operation-fact-template": HandlerPathOperationBuildFactTemplate,
	"builder-operation-fact":          HandlerPathOperationBuildFact,
	"builder-operation-sign":          HandlerPathOperationBuildSign,
	"builder-operation":               HandlerPathOperationBuild,
	"builder-send":                    HandlerPathSend,
}

var (
	UnknownProblem     = NewProblem(DefaultProblemType, "unknown problem occurred")
	unknownProblemJSON []byte
)

var GlobalItemsLimit int64 = 10

func init() {
	if b, err := jsonenc.Marshal(UnknownProblem); err != nil {
		panic(err)
	} else {
		unknownProblemJSON = b
	}
}

type Handlers struct {
	*logging.Logging
	networkID       base.NetworkID
	encs            *encoder.Encoders
	enc             encoder.Encoder
	database        *Database
	cache           Cache
	cp              *currency.CurrencyPool
	nodeInfoHandler network.NodeInfoHandler
	send            func(interface{}) (seal.Seal, error)
	router          *mux.Router
	routes          map[ /* path */ string]*mux.Route
	itemsLimiter    func(string /* request type */) int64
	rateLimit       map[string][]process.RateLimitRule
	rateLimitStore  limiter.Store
	rg              *singleflight.Group
	expireNotFilled time.Duration
}

func NewHandlers(
	networkID base.NetworkID,
	encs *encoder.Encoders,
	enc encoder.Encoder,
	st *Database,
	cache Cache,
	cp *currency.CurrencyPool,
) *Handlers {
	return &Handlers{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "http2-handlers")
		}),
		networkID:       networkID,
		encs:            encs,
		enc:             enc,
		database:        st,
		cache:           cache,
		cp:              cp,
		router:          mux.NewRouter(),
		routes:          map[string]*mux.Route{},
		itemsLimiter:    DefaultItemsLimiter,
		rateLimit:       map[string][]process.RateLimitRule{},
		rg:              &singleflight.Group{},
		expireNotFilled: time.Second * 3,
	}
}

func (hd *Handlers) Initialize() error {
	cors := handlers.CORS(
		handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"content-type"}),
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowCredentials(),
	)
	hd.router.Use(cors)

	hd.setHandlers()

	return nil
}

func (hd *Handlers) SetLimiter(f func(string) int64) *Handlers {
	hd.itemsLimiter = f

	return hd
}

func (hd *Handlers) Cache() Cache {
	return hd.cache
}

func (hd *Handlers) Router() *mux.Router {
	return hd.router
}

func (hd *Handlers) Handler() http.Handler {
	return network.HTTPLogHandler(hd.router, hd.Log())
}

func (hd *Handlers) setHandlers() {
	_ = hd.setHandler(HandlerPathCurrencies, hd.handleCurrencies, true).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathCurrency, hd.handleCurrency, true).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathManifests, hd.handleManifests, true).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathOperations, hd.handleOperations, true).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathOperation, hd.handleOperation, true).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathOperationsByHeight, hd.handleOperationsByHeight, true).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathManifestByHeight, hd.handleManifestByHeight, true).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathManifestByHash, hd.handleManifestByHash, true).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathBlockByHeight, hd.handleBlock, true).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathBlockByHash, hd.handleBlock, true).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathAccount, hd.handleAccount, true).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathAccountOperations, hd.handleAccountOperations, true).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathAccounts, hd.handleAccounts, true).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathOperationBuildFactTemplate, hd.handleOperationBuildFactTemplate, true).
		Methods(http.MethodOptions, "GET")
	_ = hd.setHandler(HandlerPathOperationBuildFact, hd.handleOperationBuildFact, false).
		Methods(http.MethodOptions, http.MethodPost)
	_ = hd.setHandler(HandlerPathOperationBuildSign, hd.handleOperationBuildSign, false).
		Methods(http.MethodOptions, http.MethodPost)
	_ = hd.setHandler(HandlerPathOperationBuild, hd.handleOperationBuild, true).
		Methods(http.MethodOptions, http.MethodGet, http.MethodPost)
	_ = hd.setHandler(HandlerPathSend, hd.handleSend, false).
		Methods(http.MethodOptions, http.MethodPost)
	_ = hd.setHandler(HandlerPathNodeInfo, hd.handleNodeInfo, true).
		Methods(http.MethodOptions, "GET")
}

func (hd *Handlers) setHandler(prefix string, h network.HTTPHandlerFunc, useCache bool) *mux.Route {
	var handler http.Handler
	if !useCache {
		handler = http.HandlerFunc(h)
	} else {
		ch := NewCachedHTTPHandler(hd.cache, h)
		_ = ch.SetLogging(hd.Logging)

		handler = ch
	}

	var name string
	if prefix == "" || prefix == "/" {
		name = "root"
	} else {
		name = prefix
	}

	var route *mux.Route
	if r := hd.router.Get(name); r != nil {
		route = r
	} else {
		route = hd.router.Name(name)
	}

	if rules, found := hd.rateLimit[prefix]; found {
		handler = process.NewRateLimitMiddleware(
			process.NewRateLimit(rules, limiter.Rate{Limit: -1}), // NOTE by default, unlimited
			hd.rateLimitStore,
		).Middleware(handler)

		hd.Log().Debug().Str("prefix", prefix).Msg("ratelimit middleware attached")
	}

	route = route.
		Path(prefix).
		Handler(handler)

	hd.routes[prefix] = route

	return route
}

func (hd *Handlers) combineURL(path string, pairs ...string) (string, error) {
	if n := len(pairs); n%2 != 0 {
		return "", errors.Errorf("failed to combine url; uneven pairs to combine url")
	} else if n < 1 {
		u, err := hd.routes[path].URL()
		if err != nil {
			return "", errors.Wrap(err, "failed to combine url")
		}
		return u.String(), nil
	}

	u, err := hd.routes[path].URLPath(pairs...)
	if err != nil {
		return "", errors.Wrap(err, "failed to combine url")
	}
	return u.String(), nil
}

func (hd *Handlers) SetRateLimit(rules map[string][]process.RateLimitRule, store limiter.Store) *Handlers {
	hd.rateLimit = rules
	hd.rateLimitStore = store

	return hd
}

func CacheKeyPath(r *http.Request) string {
	return r.URL.Path
}

func CacheKey(key string, s ...string) string {
	return fmt.Sprintf("%s-%s", key, strings.Join(s, ","))
}

func DefaultItemsLimiter(string) int64 {
	return GlobalItemsLimit
}
