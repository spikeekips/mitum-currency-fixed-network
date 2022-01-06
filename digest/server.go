package digest

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/net/http2"
)

type HTTP2Server struct {
	sync.RWMutex
	*logging.Logging
	*util.ContextDaemon
	bind             string
	host             string
	srv              *http.Server
	idleTimeout      time.Duration
	activeTimeout    time.Duration
	keepAliveTimeout time.Duration
	router           *mux.Router
}

func NewHTTP2Server(bind, host string, certs []tls.Certificate) (*HTTP2Server, error) {
	if err := network.CheckBindIsOpen("tcp", bind, time.Second*1); err != nil {
		return nil, errors.Wrap(err, "failed to open digest server")
	}

	idleTimeout := time.Second * 10
	sv := &HTTP2Server{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "http2-server")
		}),
		bind:             bind,
		host:             host,
		idleTimeout:      idleTimeout,     // TODO can be configurable
		activeTimeout:    time.Minute * 1, // TODO can be configurable
		keepAliveTimeout: time.Minute * 1, // TODO can be configurable
		router:           mux.NewRouter(),
	}

	srv, err := newHTTP2Server(sv, certs)
	if err != nil {
		return nil, err
	}
	sv.srv = srv

	sv.ContextDaemon = util.NewContextDaemon("http2-server", sv.start)

	return sv, nil
}

func newHTTP2Server(sv *HTTP2Server, certs []tls.Certificate) (*http.Server, error) {
	srv := &http.Server{
		Addr:         sv.bind,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Minute * 1,
		IdleTimeout:  sv.idleTimeout,
		TLSConfig: &tls.Config{
			Certificates: certs,
			MinVersion:   tls.VersionTLS12,
		},
		// ErrorLog:  // TODO connect with http logging
		Handler: network.HTTPLogHandler(sv.router, sv.Log()),
	}
	if err := http2.ConfigureServer(srv, &http2.Server{
		NewWriteScheduler: func() http2.WriteScheduler {
			return http2.NewPriorityWriteScheduler(nil)
		},
	}); err != nil {
		return nil, err
	}
	return srv, nil
}

func (sv *HTTP2Server) Initialize() error {
	if ln, err := net.Listen("tcp", sv.bind); err != nil {
		return err
	} else if err := ln.Close(); err != nil {
		return err
	}

	root := sv.router.Name("root")
	root.Path("/").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		},
	)

	return nil
}

func (sv *HTTP2Server) SetLogging(l *logging.Logging) *logging.Logging {
	_ = sv.ContextDaemon.SetLogging(l)

	return sv.Logging.SetLogging(l)
}

func (sv *HTTP2Server) Router() *mux.Router {
	sv.RLock()
	defer sv.RUnlock()

	return sv.router
}

func (sv *HTTP2Server) SetRouter(router *mux.Router) {
	sv.Lock()
	defer sv.Unlock()

	sv.router = router

	sv.srv.Handler = router
}

func (sv *HTTP2Server) start(ctx context.Context) error {
	ln, err := net.Listen("tcp", sv.bind)
	if err != nil {
		return err
	}
	sv.srv.Handler = network.HTTPLogHandler(sv.srv.Handler, sv.Log())

	var listener net.Listener = tcpKeepAliveListener{
		TCPListener:      ln.(*net.TCPListener),
		keepAliveTimeout: sv.keepAliveTimeout,
	}

	if len(sv.srv.TLSConfig.Certificates) > 0 {
		listener = tls.NewListener(listener, sv.srv.TLSConfig)
	}

	errchan := make(chan error)
	sv.srv.ConnState = sv.idleTimeoutHook()
	go func() {
		errchan <- sv.srv.Serve(listener)
	}()

	select {
	case err := <-errchan:
		if err != nil && errors.Is(err, http.ErrServerClosed) {
			sv.Log().Debug().Msg("server closed")

			return nil
		}

		sv.Log().Error().Err(err).Msg("something wrong")

		return err
	case <-ctx.Done():
		nctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		return sv.srv.Shutdown(nctx) // nolint:contextcheck
	default:
		return nil
	}
}

func (sv *HTTP2Server) idleTimeoutHook() func(net.Conn, http.ConnState) {
	var mu sync.Mutex
	m := map[net.Conn]*time.Timer{}
	return func(c net.Conn, cs http.ConnState) {
		mu.Lock()
		defer mu.Unlock()
		if t, ok := m[c]; ok {
			delete(m, c)
			t.Stop()
		}
		var d time.Duration
		switch cs {
		case http.StateIdle:
			d = sv.idleTimeout
		case http.StateNew, http.StateActive:
			d = sv.activeTimeout
		default:
			return
		}
		m[c] = time.AfterFunc(d, func() {
			sv.Log().Debug().
				Dur("idle-timeout", d).
				Str("remote", c.RemoteAddr().String()).
				Msg("closing idle conn after timeout")

			go func() {
				_ = c.Close()
			}()
		})
	}
}

type tcpKeepAliveListener struct {
	*net.TCPListener
	keepAliveTimeout time.Duration
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	if err := tc.SetKeepAlive(true); err != nil {
		return nil, err
	}

	if err := tc.SetKeepAlivePeriod(ln.keepAliveTimeout); err != nil {
		return nil, err
	}

	return tc, nil
}
