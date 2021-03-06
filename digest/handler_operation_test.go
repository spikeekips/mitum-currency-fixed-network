// +build mongodb

package digest

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testHandlerOperation struct {
	baseTestHandlers
}

func (t *testHandlerOperation) TestNew() {
	st, _ := t.Storage()

	var vas []OperationValue
	for i := 0; i < 10; i++ {
		sender := currency.MustAddress(util.UUID().String())
		tf := t.newTransfer(sender, currency.MustAddress(util.UUID().String()))
		doc, err := NewOperationDoc(tf, t.BSONEnc, base.Height(i), localtime.Now(), true, uint64(i))
		t.NoError(err)
		_ = t.insertDoc(st, defaultColNameOperation, doc)

		vas = append(vas, doc.va)
	}

	handlers := t.handlers(st, DummyCache{})

	for _, va := range vas {
		self, err := handlers.router.Get(HandlerPathOperation).URLPath("hash", va.Operation().Fact().Hash().String())
		t.NoError(err)

		w := t.requestOK(handlers, "GET", self.String(), nil)

		b, err := io.ReadAll(w.Result().Body)
		t.NoError(err)

		hal := t.loadHal(b)

		// NOTE check self link
		t.Equal(self.String(), hal.Links()["self"].Href())

		var uva OperationValue
		t.NoError(t.JSONEnc.Decode(hal.RawInterface(), &uva))

		t.Equal(va.Height(), uva.Height())
		t.compareOperationValue(va, uva)
	}
}

func (t *testHandlerOperation) TestNotFound() {
	st, _ := t.Storage()

	handlers := t.handlers(st, DummyCache{})

	self, err := handlers.router.Get(HandlerPathOperation).URLPath("hash", valuehash.RandomSHA256().String())
	t.NoError(err)

	w := t.request404(handlers, "GET", self.String(), nil)

	b, err := io.ReadAll(w.Result().Body)
	t.NoError(err)

	var problem Problem
	t.NoError(jsonenc.Unmarshal(b, &problem))
	t.Contains(problem.Error(), "operation not found")
}

func TestHandlerOperation(t *testing.T) {
	suite.Run(t, new(testHandlerOperation))
}

type testHandlerOperations struct {
	baseTestHandlers
}

func (t *testHandlerOperations) TestOperationsPaging() {
	st, _ := t.Storage()

	var hashes []string

	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			height := base.Height(i % 3)
			index := uint64(j)
			tf := t.newTransfer(currency.MustAddress(util.UUID().String()), currency.MustAddress(util.UUID().String()))
			doc, err := NewOperationDoc(tf, t.BSONEnc, height, localtime.Now(), true, index)
			t.NoError(err)
			_ = t.insertDoc(st, defaultColNameOperation, doc)

			fh := tf.Fact().Hash().String()

			hashes = append(hashes, fh)
		}
	}

	var limit int64 = 3
	handlers := t.handlers(st, DummyCache{})
	_ = handlers.SetLimiter(func(string) int64 {
		return limit
	})

	{ // no reverse
		reverse := false
		offset := ""

		self, err := handlers.router.Get(HandlerPathOperations).URL()
		t.NoError(err)
		self.RawQuery = fmt.Sprintf("%s&%s", stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))

		var uhashes []string
		for {
			w := t.request(handlers, "GET", self.String(), nil)

			if r := w.Result().StatusCode; r == http.StatusOK {
				t.Equal(HALMimetype, w.Result().Header.Get("content-type"))
				t.Equal(handlers.enc.Hint().String(), w.Result().Header.Get(HTTP2EncoderHintHeader))
			} else if r == http.StatusNotFound {
				break
			}

			b, err := io.ReadAll(w.Result().Body)
			t.NoError(err)

			hal := t.loadHal(b)

			var em []BaseHal
			t.NoError(jsonenc.Unmarshal(hal.RawInterface(), &em))
			t.True(int(limit) >= len(em))

			for _, b := range em {
				var va OperationValue
				t.NoError(t.JSONEnc.Decode(b.RawInterface(), &va))
				fh := va.Operation().Fact().Hash().String()
				uhashes = append(uhashes, fh)
			}

			next, err := hal.Links()["next"].URL()
			t.NoError(err)
			self = next

			if int64(len(em)) < limit {
				break
			}
		}

		t.Equal(hashes, uhashes)
	}

	{ // reverse
		var rhashes []string
		for i := len(hashes) - 1; i >= 0; i-- {
			rhashes = append(rhashes, hashes[i])
		}

		reverse := true
		offset := ""

		self, err := handlers.router.Get(HandlerPathOperations).URL()
		t.NoError(err)
		self.RawQuery = fmt.Sprintf("%s&%s", stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))

		var uhashes []string
		for {
			w := t.request(handlers, "GET", self.String(), nil)
			if r := w.Result().StatusCode; r == http.StatusOK {
				t.Equal(HALMimetype, w.Result().Header.Get("content-type"))
				t.Equal(handlers.enc.Hint().String(), w.Result().Header.Get(HTTP2EncoderHintHeader))
			} else if r == http.StatusNotFound {
				break
			}

			b, err := io.ReadAll(w.Result().Body)
			t.NoError(err)

			hal := t.loadHal(b)

			var em []BaseHal
			t.NoError(jsonenc.Unmarshal(hal.RawInterface(), &em))
			t.True(int(limit) >= len(em))

			for _, b := range em {
				var va OperationValue
				t.NoError(t.JSONEnc.Decode(b.RawInterface(), &va))
				fh := va.Operation().Fact().Hash().String()
				uhashes = append(uhashes, fh)
			}

			next, err := hal.Links()["next"].URL()
			t.NoError(err)
			self = next

			if int64(len(em)) < limit {
				break
			}
		}

		t.Equal(rhashes, uhashes)
	}
}

func (t *testHandlerOperations) TestOperationsByHeightPaging() {
	st, _ := t.Storage()

	hashesByHeight := map[base.Height][]string{}

	for i := 0; i < 3; i++ {
		height := base.Height(i)
		var hs []string
		for j := 0; j < 3; j++ {
			index := uint64(j)
			tf := t.newTransfer(currency.MustAddress(util.UUID().String()), currency.MustAddress(util.UUID().String()))
			doc, err := NewOperationDoc(tf, t.BSONEnc, height, localtime.Now(), true, index)
			t.NoError(err)
			_ = t.insertDoc(st, defaultColNameOperation, doc)

			fh := tf.Fact().Hash().String()

			hs = append(hs, fh)
		}

		hashesByHeight[height] = hs
	}

	var limit int64 = 3
	handlers := t.handlers(st, DummyCache{})
	_ = handlers.SetLimiter(func(string) int64 {
		return limit
	})

	{ // no reverse
		height := base.Height(1)
		reverse := false
		offset := ""

		self, err := handlers.router.Get(HandlerPathOperationsByHeight).URLPath("height", height.String())
		t.NoError(err)
		self.RawQuery = fmt.Sprintf("%s&%s", stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))

		var uhashes []string
		for {
			w := t.request(handlers, "GET", self.String(), nil)

			if r := w.Result().StatusCode; r == http.StatusOK {
				t.Equal(HALMimetype, w.Result().Header.Get("content-type"))
				t.Equal(handlers.enc.Hint().String(), w.Result().Header.Get(HTTP2EncoderHintHeader))
			} else if r == http.StatusNotFound {
				break
			}

			b, err := io.ReadAll(w.Result().Body)
			t.NoError(err)

			hal := t.loadHal(b)

			var em []BaseHal
			t.NoError(jsonenc.Unmarshal(hal.RawInterface(), &em))
			t.True(int(limit) >= len(em))

			for _, b := range em {
				var va OperationValue
				t.NoError(t.JSONEnc.Decode(b.RawInterface(), &va))
				fh := va.Operation().Fact().Hash().String()
				uhashes = append(uhashes, fh)
			}

			next, err := hal.Links()["next"].URL()
			t.NoError(err)
			self = next

			if int64(len(em)) < limit {
				break
			}
		}

		t.Equal(hashesByHeight[height], uhashes)
	}

	{ // reverse
		height := base.Height(1)
		var rhashes []string
		for i := len(hashesByHeight[height]) - 1; i >= 0; i-- {
			rhashes = append(rhashes, hashesByHeight[height][i])
		}

		reverse := true
		offset := ""

		self, err := handlers.router.Get(HandlerPathOperationsByHeight).URLPath("height", height.String())
		t.NoError(err)
		self.RawQuery = fmt.Sprintf("%s&%s", stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))

		var uhashes []string
		for {
			w := t.request(handlers, "GET", self.String(), nil)
			if r := w.Result().StatusCode; r == http.StatusOK {
				t.Equal(HALMimetype, w.Result().Header.Get("content-type"))
				t.Equal(handlers.enc.Hint().String(), w.Result().Header.Get(HTTP2EncoderHintHeader))
			} else if r == http.StatusNotFound {
				break
			}

			b, err := io.ReadAll(w.Result().Body)
			t.NoError(err)

			hal := t.loadHal(b)

			var em []BaseHal
			t.NoError(jsonenc.Unmarshal(hal.RawInterface(), &em))
			t.True(int(limit) >= len(em))

			for _, b := range em {
				var va OperationValue
				t.NoError(t.JSONEnc.Decode(b.RawInterface(), &va))
				fh := va.Operation().Fact().Hash().String()
				uhashes = append(uhashes, fh)
			}

			next, err := hal.Links()["next"].URL()
			t.NoError(err)
			self = next

			if int64(len(em)) < limit {
				break
			}
		}

		t.Equal(rhashes, uhashes)
	}
}

func TestHandlerOperations(t *testing.T) {
	suite.Run(t, new(testHandlerOperations))
}
