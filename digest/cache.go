package digest

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"net/textproto"
	"net/url"
	"time"

	"github.com/bluele/gcache"
	"github.com/rainycape/memcache"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/errors"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

var (
	DefaultCacheExpire = time.Hour
	SkipCacheError     = errors.NewError("skip cache")
)

type Cache interface {
	Get(string) ([]byte, error)
	Set(string, []byte, time.Duration) error
}

func NewCacheFromURI(uri string) (Cache, error) {
	if u, err := url.Parse(uri); err != nil {
		return nil, xerrors.Errorf("invalid uri of cache, %q: %w", uri, err)
	} else {
		switch {
		case u.Scheme == "memory":
			// TODO set size, expire
			return NewLocalMemCache(100*100, time.Second*10), nil
		case u.Scheme == "memcached":
			return NewMemcached(u.Host)
		default:
			return nil, xerrors.Errorf("unsupported uri of cache, %q", uri)
		}
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
	if i, err := ca.cl.Get(key); err != nil {
		return nil, err
	} else {
		return i.([]byte), nil
	}
}

func (ca *LocalMemCache) Set(key string, b []byte, expire time.Duration) error {
	return ca.cl.SetWithExpire(key, b, expire)
}

type Memcached struct {
	cl *memcache.Client
}

func NewMemcached(servers ...string) (*Memcached, error) {
	if cl, err := memcache.New(servers...); err != nil {
		return nil, err
	} else {
		if _, err := cl.Get("<any key>"); err != nil {
			if !xerrors.Is(err, memcache.ErrCacheMiss) {
				return nil, err
			}
		}

		return &Memcached{cl: cl}, nil
	}
}

func (mc *Memcached) Get(key string) ([]byte, error) {
	if item, err := mc.cl.Get(key); err != nil {
		return nil, err
	} else {
		return item.Value, nil
	}
}

func (mc *Memcached) Set(key string, b []byte, expire time.Duration) error {
	return mc.cl.Set(&memcache.Item{Key: key, Value: b, Expiration: int32(expire.Seconds())})
}

type DummyCache struct {
}

func (ca DummyCache) Get(string) ([]byte, error) {
	return nil, storage.NotFoundError
}

func (ca DummyCache) Set(string, []byte, time.Duration) error {
	return nil
}

type CachedHTTPHandler struct {
	*logging.Logging
	cache Cache
	f     func(http.ResponseWriter, *http.Request)
}

func NewCachedHTTPHandler(cache Cache, f func(http.ResponseWriter, *http.Request)) CachedHTTPHandler {
	return CachedHTTPHandler{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
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
		if !xerrors.Is(err, SkipCacheError) {
			ch.Log().Verbose().Err(err).Msg("failed to cache")
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
		return makeCacheKey(cr.key)
	}

	return cacheKeyFromRequest(cr.r)
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
	} else {
		_, _ = buf.Write([]byte{'\r', '\n'})
		_, _ = buf.Write(cr.buf.Bytes())

		return cr.cache.Set(cr.Key(), buf.Bytes(), cr.Expire())
	}
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

func loadFromCache(cache Cache, key string, w http.ResponseWriter) error {
	if b, err := cache.Get(cacheKey(key)); err != nil {
		return err
	} else if err = writeFromCache(b, w); err != nil {
		return err
	} else {
		return nil
	}
}

func writeFromCache(b []byte, w http.ResponseWriter) error {
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
			if hr, err := tp.ReadMIMEHeader(); err != nil {
				return err
			} else {
				for k := range hr {
					w.Header().Set(k, hr.Get(k))
				}
			}

			wroteHeader = true
			continue
		} else {
			_, _ = w.Write(sb)
		}
	}

	if cw, ok := w.(*CacheResponseWriter); ok {
		_ = cw.SkipCache()
	}

	return nil
}

func makeCacheKey(key string) string {
	return valuehash.NewSHA256([]byte(key)).String()
}

func cacheKeyFromRequest(r *http.Request) string {
	return makeCacheKey(r.URL.Path + "?" + r.URL.Query().Encode())
}
