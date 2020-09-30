// +build mongodb

package digest

import (
	"net/http"
	"net/http/httptest"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type baseTestHandlers struct {
	baseTest
}

func (t *baseTestHandlers) handlers(st *Storage, cache Cache) *Handlers {
	handlers := NewHandlers(t.JSONEnc, st, cache)
	t.NoError(handlers.Initialize())

	return handlers
}

func (t *baseTestHandlers) request(handlers *Handlers, method, path string) *httptest.ResponseRecorder {
	r, err := http.NewRequest(method, "http://localhost"+path, nil)
	t.NoError(err)

	w := httptest.NewRecorder()
	handlers.Handler().ServeHTTP(w, r)

	return w
}

func (t *baseTestHandlers) requestOK(handlers *Handlers, method, path string) *httptest.ResponseRecorder {
	w := t.request(handlers, method, path)

	t.Equal(http.StatusOK, w.Result().StatusCode)
	t.Equal(HALMimetype, w.Result().Header.Get("content-type"))
	t.Equal(handlers.enc.Hint().String(), w.Result().Header.Get(HTTP2EncoderHintHeader))

	return w
}

func (t *baseTestHandlers) request404(handlers *Handlers, method, path string) *httptest.ResponseRecorder {
	w := t.request(handlers, method, path)

	t.Equal(http.StatusNotFound, w.Result().StatusCode)
	t.Equal(ProblemMimetype, w.Result().Header.Get("content-type"))
	t.Equal(handlers.enc.Hint().String(), w.Result().Header.Get(HTTP2EncoderHintHeader))

	return w
}

func (t *baseTestHandlers) loadHal(b []byte) BaseHal {
	var m BaseHal
	t.NoError(jsonenc.Unmarshal(b, &m))

	return m
}
