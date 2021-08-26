package digest

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"net/textproto"
	"time"

	"github.com/bluele/gcache"
	"github.com/pkg/errors"
	"github.com/rainycape/memcache"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	DefaultCacheExpire = time.Hour
	SkipCacheError     = util.NewError("skip cache")
)

type Cache interface {
	Get(string) ([]byte, error)
	Set(string, []byte, time.Duration) error
}

func NewCacheFromURI(uri string) (Cache, error) {
	u, err := network.ParseURL(uri, false)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid uri of cache, %q", uri)
	}
	switch {
	case u.Scheme == "memory":
		// TODO set size, expire
		return NewLocalMemCache(100*100, time.Second*10), nil
	case u.Scheme == "memcached":
		return NewMemcached(u.Host)
	default:
		return nil, errors.Errorf("unsupported uri of cache, %q", uri)
	}
}

type LocalMemCache struct {
	cl gcache.Cache
}

func NewLocalMemCache(size int, expire time.Duration) *LocalMemCache {
	cl := gcache.New(size).LRU().
		Expiration(expire).
		Build()

	return &LocalMemCache{cl: cl}
}

func (ca *LocalMemCache) Get(key string) ([]byte, error) {
	i, err := ca.cl.Get(key)
	if err != nil {
		return nil, err
	}
	return i.([]byte), nil
}

func (ca *LocalMemCache) Set(key string, b []byte, expire time.Duration) error {
	return ca.cl.SetWithExpire(key, b, expire)
}

type Memcached struct {
	cl *memcache.Client
}

func NewMemcached(servers ...string) (*Memcached, error) {
	cl, err := memcache.New(servers...)
	if err != nil {
		return nil, err
	}
	if _, err := cl.Get("<any key>"); err != nil {
		if !errors.Is(err, memcache.ErrCacheMiss) {
			return nil, err
		}
	}

	return &Memcached{cl: cl}, nil
}

func (mc *Memcached) Get(key string) ([]byte, error) {
	item, err := mc.cl.Get(key)
	if err != nil {
		return nil, err
	}
	return item.Value, nil
}

func (mc *Memcached) Set(key string, b []byte, expire time.Duration) error {
	return mc.cl.Set(&memcache.Item{Key: key, Value: b, Expiration: int32(expire.Seconds())})
}

type DummyCache struct{}

func (DummyCache) Get(string) ([]byte, error) {
	return nil, util.NotFoundError
}

func (DummyCache) Set(string, []byte, time.Duration) error {
	return nil
}

type CachedHTTPHandler struct {
	*logging.Logging
	cache Cache
	f     func(http.ResponseWriter, *http.Request)
}

func NewCachedHTTPHandler(cache Cache, f func(http.ResponseWriter, *http.Request)) CachedHTTPHandler {
	return CachedHTTPHandler{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "cached-http-handler")
		}),
		cache: cache,
		f:     f,
	}
}

func (ch CachedHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cr := NewCacheResponseWriter(ch.cache, w, r)

	ch.f(cr, r)

	if err := cr.Cache(); err != nil {
		if !errors.Is(err, SkipCacheError) {
			ch.Log().Debug().Err(err).Msg("failed to cache")
		}
	}
}

type CacheResponseWriter struct {
	http.ResponseWriter
	cache     Cache
	r         *http.Request
	buf       *bytes.Buffer
	status    int
	key       string
	expire    time.Duration
	skipCache bool
	writer    io.Writer
}

func NewCacheResponseWriter(cache Cache, w http.ResponseWriter, r *http.Request) *CacheResponseWriter {
	buf := &bytes.Buffer{}
	return &CacheResponseWriter{
		ResponseWriter: w,
		cache:          cache,
		r:              r,
		buf:            buf,
		status:         http.StatusOK,
		writer:         io.MultiWriter(w, buf),
	}
}

func (cr *CacheResponseWriter) Write(b []byte) (int, error) {
	return cr.writer.Write(b)
}

func (cr *CacheResponseWriter) WriteHeader(status int) {
	cr.ResponseWriter.WriteHeader(status)
	cr.status = status
}

func (cr *CacheResponseWriter) OK() bool {
	return cr.status == http.StatusOK || cr.status == http.StatusCreated || cr.status == http.StatusMovedPermanently
}

func (cr *CacheResponseWriter) filterHeader() http.Header {
	nh := http.Header{}
	for k := range cr.Header() {
		switch http.CanonicalHeaderKey(k) {
		case HTTP2EncoderHintHeader:
		case "Content-Type":
		default:
			continue
		}
		nh.Add(k, cr.Header().Get(k))
	}

	return nh
}

func (cr *CacheResponseWriter) Key() string {
	if len(cr.key) > 0 {
		return MakeCacheKey(cr.key)
	}

	return CacheKeyFromRequest(cr.r)
}

func (cr *CacheResponseWriter) SetKey(key string) *CacheResponseWriter {
	cr.key = key

	return cr
}

func (cr *CacheResponseWriter) Expire() time.Duration {
	if cr.expire > 0 {
		return cr.expire
	}

	return DefaultCacheExpire
}

func (cr *CacheResponseWriter) SetExpire(expire time.Duration) *CacheResponseWriter {
	cr.expire = expire

	return cr
}

func (cr *CacheResponseWriter) SkipCache() *CacheResponseWriter {
	cr.skipCache = true

	return cr
}

func (cr *CacheResponseWriter) Cache() error {
	if cr.skipCache {
		return SkipCacheError
	}

	if !cr.OK() {
		return nil
	}

	buf := &bytes.Buffer{}
	if err := cr.filterHeader().Write(buf); err != nil {
		return err
	}
	_, _ = buf.Write([]byte{'\r', '\n'})
	_, _ = buf.Write(cr.buf.Bytes())

	return cr.cache.Set(cr.Key(), buf.Bytes(), cr.Expire())
}

func ScanCRLF(data []byte, atEOF bool) (int, []byte, error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte{'\r', '\n', '\r', '\n'}); i >= 0 {
		return i + 4, data[0:i], nil
	}

	if atEOF {
		return len(data), data, nil
	}

	return 0, nil, nil
}

func LoadFromCache(cache Cache, key string, w http.ResponseWriter) error {
	if b, err := cache.Get(MakeCacheKey(key)); err != nil {
		return err
	} else if err = WriteFromCache(b, w); err != nil {
		return err
	} else {
		return nil
	}
}

func WriteFromCache(b []byte, w http.ResponseWriter) error {
	reader := bufio.NewReader(bytes.NewReader(b))
	sc := bufio.NewScanner(reader)
	sc.Split(ScanCRLF)

	var wroteHeader bool
	for sc.Scan() {
		sb := sc.Bytes()
		sb = append(sb, '\r', '\n')

		if !wroteHeader {
			buf := bytes.NewReader(append(sb, '\r', '\n'))
			tp := textproto.NewReader(bufio.NewReader(buf))
			hr, err := tp.ReadMIMEHeader()
			if err != nil {
				return err
			}
			for k := range hr {
				w.Header().Set(k, hr.Get(k))
			}

			wroteHeader = true
			continue
		}
		_, _ = w.Write(sb)
	}

	if cw, ok := w.(*CacheResponseWriter); ok {
		_ = cw.SkipCache()
	}

	return nil
}

func MakeCacheKey(key string) string {
	return valuehash.NewSHA256([]byte(key)).String()
}

func CacheKeyFromRequest(r *http.Request) string {
	return MakeCacheKey(r.URL.Path + "?" + r.URL.Query().Encode())
}
