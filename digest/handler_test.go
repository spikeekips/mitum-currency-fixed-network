//go:build mongodb
// +build mongodb

package digest

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type baseTestHandlers struct {
	baseTest
}

func (t *baseTestHandlers) handlers(st *Database, cache Cache) *Handlers {
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

func (t *baseTestHandlers) request400(handlers *Handlers, method, path string, data []byte) (*httptest.ResponseRecorder, Problem) {
	w := t.request(handlers, method, path, data)

	t.Equal(http.StatusBadRequest, w.Result().StatusCode)
	t.Equal(ProblemMimetype, w.Result().Header.Get("content-type"))

	b, err := io.ReadAll(w.Result().Body)
	t.NoError(err)

	hinter, err := t.JSONEnc.Decode(b)
	t.NoError(err)

	problem, ok := hinter.(Problem)
	t.True(ok)

	return w, problem
}

func (t *baseTestHandlers) request404(handlers *Handlers, method, path string, data []byte) *httptest.ResponseRecorder {
	w := t.request(handlers, method, path, data)

	t.Equal(http.StatusNotFound, w.Result().StatusCode)
	t.Equal(ProblemMimetype, w.Result().Header.Get("content-type"))

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

	hinter, err := t.JSONEnc.Decode(b)
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

func (t *baseTestHandlers) getItems(handlers *Handlers, limit int, self *url.URL, decode func([]byte) (interface{}, error)) []interface{} {
	var hs []interface{}
	for {
		w := t.request(handlers, "GET", self.String(), nil)
		if r := w.Result().StatusCode; r == http.StatusOK {
			t.Equal(HALMimetype, w.Result().Header.Get("content-type"))
			t.Equal(handlers.enc.Hint().String(), w.Result().Header.Get(HTTP2EncoderHintHeader))
		} else if r == http.StatusNotFound {
			break
		} else {
			panic(w)
		}

		b, err := io.ReadAll(w.Result().Body)
		t.NoError(err)

		hal := t.loadHal(b)

		if len(hal.RawInterface()) < 1 {
			break
		}

		var em []BaseHal
		t.NoError(jsonenc.Unmarshal(hal.RawInterface(), &em))
		t.True(limit >= len(em))

		for _, b := range em {
			h, err := decode(b.RawInterface())
			t.NoError(err)

			hs = append(hs, h)
		}

		next, err := hal.Links()["next"].URL()
		t.NoError(err)
		self = next

		if len(em) < limit {
			break
		}
	}

	return hs
}
