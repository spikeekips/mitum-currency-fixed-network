// +build mongodb

package digest

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type baseTestHandlers struct {
	baseTest
}

func (t *baseTestHandlers) handlers(st *Storage, cache Cache) *Handlers {
	handlers := NewHandlers(t.networkID, t.Encs, t.JSONEnc, st, cache, nil)
	t.NoError(handlers.Initialize())

	return handlers
}

func (t *baseTestHandlers) request(handlers *Handlers, method, path string, data []byte) *httptest.ResponseRecorder {
	var body io.Reader
	if data != nil {
		body = bytes.NewBuffer(data)
	}

	r, err := http.NewRequest(method, "http://localhost"+path, body)
	t.NoError(err)

	w := httptest.NewRecorder()
	handlers.Handler().ServeHTTP(w, r)

	return w
}

func (t *baseTestHandlers) requestOK(handlers *Handlers, method, path string, data []byte) *httptest.ResponseRecorder {
	w := t.request(handlers, method, path, data)

	t.Equal(http.StatusOK, w.Result().StatusCode)
	t.Equal(HALMimetype, w.Result().Header.Get("content-type"))
	t.Equal(handlers.enc.Hint().String(), w.Result().Header.Get(HTTP2EncoderHintHeader))

	if w.Result().StatusCode != http.StatusOK {
		panic(w)
	}

	return w
}

func (t *baseTestHandlers) request404(handlers *Handlers, method, path string, data []byte) *httptest.ResponseRecorder {
	w := t.request(handlers, method, path, data)

	t.Equal(http.StatusNotFound, w.Result().StatusCode)
	t.Equal(ProblemMimetype, w.Result().Header.Get("content-type"))
	t.Equal(handlers.enc.Hint().String(), w.Result().Header.Get(HTTP2EncoderHintHeader))

	return w
}

func (t *baseTestHandlers) request405(handlers *Handlers, method, path string, data []byte) *httptest.ResponseRecorder {
	w := t.request(handlers, method, path, data)

	t.Equal(http.StatusMethodNotAllowed, w.Result().StatusCode)

	return w
}

func (t *baseTestHandlers) request500(handlers *Handlers, method, path string, data []byte) (*httptest.ResponseRecorder, Problem) {
	w := t.request(handlers, method, path, data)

	t.Equal(http.StatusInternalServerError, w.Result().StatusCode)
	t.Equal(ProblemMimetype, w.Result().Header.Get("content-type"))

	b, err := io.ReadAll(w.Result().Body)
	t.NoError(err)

	hinter, err := t.JSONEnc.DecodeByHint(b)
	t.NoError(err)

	problem, ok := hinter.(Problem)
	t.True(ok)

	return w, problem
}

func (t *baseTestHandlers) loadHal(b []byte) BaseHal {
	var m BaseHal
	t.NoError(jsonenc.Unmarshal(b, &m))

	return m
}
