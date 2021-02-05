package digest

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	jsoniter "github.com/json-iterator/go"
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/middleware/stdlib"
	"golang.org/x/xerrors"
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
	HandlerPathAccount                    = `/account/{address:(?i)[0-9a-z][0-9a-z\-]+\-[a-z0-9]{4}\:[a-z0-9\.]*}`
	HandlerPathAccountOperations          = `/account/{address:(?i)[0-9a-z][0-9a-z\-]+\-[a-z0-9]{4}\:[a-z0-9\.]*}/operations` // nolint:lll
	HandlerPathOperationBuildFactTemplate = `/builder/operation/fact/template/{fact:[\w][\w\-]*}`
	HandlerPathOperationBuildFact         = `/builder/operation/fact`
	HandlerPathOperationBuildSign         = `/builder/operation/sign`
	HandlerPathOperationBuild             = `/builder/operation`
	HandlerPathSend                       = `/builder/send`
)

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
	storage         *Storage
	cache           Cache
	cp              *currency.CurrencyPool
	nodeInfoHandler network.NodeInfoHandler
	send            func(interface{}) (seal.Seal, error)
	router          *mux.Router
	routes          map[ /* path */ string]*mux.Route
	itemsLimiter    func(string /* request type */) int64
	rateLimiter     *limiter.Limiter
}

func NewHandlers(
	networkID base.NetworkID,
	encs *encoder.Encoders,
	enc encoder.Encoder,
	st *Storage,
	cache Cache,
	cp *currency.CurrencyPool,
) *Handlers {
	return &Handlers{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "http2-handlers")
		}),
		networkID:    networkID,
		encs:         encs,
		enc:          enc,
		storage:      st,
		cache:        cache,
		cp:           cp,
		router:       mux.NewRouter(),
		routes:       map[string]*mux.Route{},
		itemsLimiter: defaultItemsLimiter,
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
		_ = ch.SetLogger(hd.Log())

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

	if hd.rateLimiter != nil {
		handler = stdlib.NewMiddleware(hd.rateLimiter).Handler(handler)
	}

	route = route.
		Path(prefix).
		Handler(handler)

	hd.routes[prefix] = route

	return route
}

func (hd *Handlers) stream(w http.ResponseWriter, bufsize int, status int) (*jsoniter.Stream, func()) {
	w.Header().Set(HTTP2EncoderHintHeader, hd.enc.Hint().String())
	w.Header().Set("Content-Type", HALMimetype)

	if status != http.StatusOK {
		w.WriteHeader(status)
	}

	stream := jsoniter.NewStream(HALJSONConfigDefault, w, bufsize)
	return stream, func() {
		if err := stream.Flush(); err != nil {
			hd.Log().Error().Err(err).Msg("failed to straem thru jsoniterator")

			hd.problemWithError(w, err, http.StatusInternalServerError)
		}
	}
}

func (hd *Handlers) combineURL(path string, pairs ...string) (string, error) {
	if n := len(pairs); n%2 != 0 {
		return "", xerrors.Errorf("failed to combine url; uneven pairs to combine url")
	} else if n < 1 {
		if u, err := hd.routes[path].URL(); err != nil {
			return "", xerrors.Errorf("failed to combine url: %w", err)
		} else {
			return u.String(), nil
		}
	}

	if u, err := hd.routes[path].URLPath(pairs...); err != nil {
		return "", xerrors.Errorf("failed to combine url: %w", err)
	} else {
		return u.String(), nil
	}
}

func (hd *Handlers) notSupported(w http.ResponseWriter, err error) {
	if err == nil {
		err = xerrors.Errorf("not supported")
	}

	hd.problemWithError(w, err, http.StatusInternalServerError)
}

func (hd *Handlers) problemWithError(w http.ResponseWriter, err error, status int) {
	hd.writePoblem(w, NewProblemFromError(err), status)
}

func (hd *Handlers) writePoblem(w http.ResponseWriter, pr Problem, status int) {
	if status == 0 {
		status = http.StatusInternalServerError
	}

	w.Header().Set(HTTP2EncoderHintHeader, hd.enc.Hint().String())
	w.Header().Set("Content-Type", ProblemMimetype)
	w.Header().Set("X-Content-Type-Options", "nosniff")

	var output []byte
	if b, err := jsonenc.Marshal(pr); err != nil {
		hd.Log().Error().Err(err).Interface("problem", pr).Msg("failed to marshal problem, UnknownProblem will be used")

		output = unknownProblemJSON
	} else {
		output = b
	}

	w.WriteHeader(status)
	_, _ = w.Write(output)
}

func (hd *Handlers) writeHal(w http.ResponseWriter, hal Hal, status int) { // nolint:unparam
	stream, flush := hd.stream(w, 1, status)
	defer flush()

	stream.WriteVal(hal)
}

func (hd *Handlers) writeCache(w http.ResponseWriter, key string, expire time.Duration) {
	if cw, ok := w.(*CacheResponseWriter); ok {
		_ = cw.SetKey(key).SetExpire(expire)
	}
}

func (hd *Handlers) SetRateLimiter(limiter *limiter.Limiter) *Handlers {
	hd.rateLimiter = limiter

	return hd
}

func cacheKeyPath(r *http.Request) string {
	return r.URL.Path
}

func cacheKey(key string, s ...string) string {
	return fmt.Sprintf("%s-%s", key, strings.Join(s, ","))
}

func defaultItemsLimiter(string) int64 {
	return GlobalItemsLimit
}
