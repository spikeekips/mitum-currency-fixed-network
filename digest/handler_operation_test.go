//go:build mongodb
// +build mongodb

package digest

import (
	"fmt"
	"io"
	"net/url"
	"testing"
	"time"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
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
	st, _ := t.Database()

	var vas []OperationValue
	hasReasons := map[string]OperationValue{}
	for i := 0; i < 10; i++ {
		sender := currency.MustAddress(util.UUID().String())
		tf := t.newTransfer(sender, currency.MustAddress(util.UUID().String()))

		var reason operation.ReasonError
		var inState bool = true
		if i%2 == 0 {
			reason = operation.NewBaseReasonError("showme %d", i).SetData(map[string]interface{}{"i": float64(i)})
			inState = false
		}

		doc, err := NewOperationDoc(tf, t.BSONEnc, base.Height(i), localtime.UTCNow(), inState, reason, uint64(i))
		t.NoError(err)
		_ = t.insertDoc(st, defaultColNameOperation, doc)

		if i%2 == 0 {
			hasReasons[tf.Fact().Hash().String()] = doc.va
		}

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

		hinter, err := t.JSONEnc.Decode(hal.RawInterface())
		t.NoError(err)
		uva := hinter.(OperationValue)

		t.Equal(va.Height(), uva.Height())
		t.compareOperationValue(va, uva)

		ar := uva.Reason()
		ai := uva.InState()

		var br operation.ReasonError
		var bi bool = true
		if j, found := hasReasons[uva.Operation().Fact().Hash().String()]; found {
			br = j.Reason()
			bi = j.InState()

			t.Equal(ar.Msg(), br.Msg())
			t.Equal(ar.Data(), br.Data())
		} else {
			t.Nil(ar)
			t.Nil(br)
		}

		t.Equal(ai, bi)
	}
}

func (t *testHandlerOperation) TestNotFound() {
	st, _ := t.Database()

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

func (t *testHandlerOperations) getHashes(handlers *Handlers, limit int, self *url.URL) []string {
	l := t.getItems(handlers, limit, self, func(b []byte) (interface{}, error) {
		hinter, err := t.JSONEnc.Decode(b)
		if err != nil {
			return "", err
		}

		va := hinter.(OperationValue)

		return va.Operation().Fact().Hash().String(), nil
	})

	uhashes := make([]string, len(l))
	for i := range l {
		uhashes[i] = l[i].(string)
	}

	return uhashes
}

func (t *testHandlerOperations) TestOperationsPaging() {
	st, _ := t.Database()

	var hashes []string

	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			height := base.Height(i % 3)
			index := uint64(j)
			tf := t.newTransfer(currency.MustAddress(util.UUID().String()), currency.MustAddress(util.UUID().String()))
			doc, err := NewOperationDoc(tf, t.BSONEnc, height, localtime.UTCNow(), true, nil, index)
			t.NoError(err)
			_ = t.insertDoc(st, defaultColNameOperation, doc)

			fh := tf.Fact().Hash().String()

			hashes = append(hashes, fh)
		}
	}

	var limit int64 = 3
	handlers := t.handlers(st, NewLocalMemCache(1000, time.Minute))
	_ = handlers.SetLimiter(func(string) int64 {
		return limit
	})
	handlers.expireNotFilled = time.Second

	{ // no reverse
		reverse := false
		offset := ""

		self, err := handlers.router.Get(HandlerPathOperations).URL()
		t.NoError(err)
		self.RawQuery = fmt.Sprintf("%s&%s", stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))

		uhashes := t.getHashes(handlers, int(limit), self)
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

		uhashes := t.getHashes(handlers, int(limit), self)
		t.Equal(rhashes, uhashes)
	}

	t.T().Log("insert more")

	height := base.Height(4)
	for i := uint64(0); i < 3; i++ {
		tf := t.newTransfer(currency.MustAddress(util.UUID().String()), currency.MustAddress(util.UUID().String()))
		doc, err := NewOperationDoc(tf, t.BSONEnc, height, localtime.UTCNow(), true, nil, i)
		t.NoError(err)
		_ = t.insertDoc(st, defaultColNameOperation, doc)

		fh := tf.Fact().Hash().String()

		hashes = append(hashes, fh)
	}

	<-time.After(handlers.expireNotFilled + time.Millisecond) // wait empty offset expire
	{                                                         // reverse again
		var rhashes []string
		for i := len(hashes) - 1; i >= 0; i-- {
			rhashes = append(rhashes, hashes[i])
		}

		reverse := true
		offset := ""

		self, err := handlers.router.Get(HandlerPathOperations).URL()
		t.NoError(err)
		self.RawQuery = fmt.Sprintf("%s&%s", stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))

		uhashes := t.getHashes(handlers, int(limit), self)
		t.Equal(rhashes, uhashes)
	}
}

func (t *testHandlerOperations) TestOperationsByHeightPaging() {
	st, _ := t.Database()

	hashesByHeight := map[base.Height][]string{}

	for i := 0; i < 3; i++ {
		height := base.Height(i)
		var hs []string
		for j := 0; j < 3; j++ {
			index := uint64(j)
			tf := t.newTransfer(currency.MustAddress(util.UUID().String()), currency.MustAddress(util.UUID().String()))
			doc, err := NewOperationDoc(tf, t.BSONEnc, height, localtime.UTCNow(), true, nil, index)
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

		uhashes := t.getHashes(handlers, int(limit), self)
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

		uhashes := t.getHashes(handlers, int(limit), self)
		t.Equal(rhashes, uhashes)
	}
}

func TestHandlerOperations(t *testing.T) {
	suite.Run(t, new(testHandlerOperations))
}
