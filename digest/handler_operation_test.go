// +build mongodb

package digest

import (
	"io/ioutil"
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
		doc, err := NewOperationDoc(tf, t.BSONEnc, base.Height(i), localtime.Now(), true)
		t.NoError(err)
		_ = t.insertDoc(st, defaultColNameOperation, doc)

		vas = append(vas, doc.va)
	}

	handlers := t.handlers(st, DummyCache{})

	for _, va := range vas {
		self, err := handlers.router.Get(HandlerPathOperation).URLPath("hash", va.Operation().Fact().Hash().String())
		t.NoError(err)

		w := t.requestOK(handlers, "GET", self.String())

		b, err := ioutil.ReadAll(w.Result().Body)
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

	w := t.request404(handlers, "GET", self.String())

	b, err := ioutil.ReadAll(w.Result().Body)
	t.NoError(err)

	var problem Problem
	t.NoError(jsonenc.Unmarshal(b, &problem))
	t.Contains(problem.Error(), "operation not found")
}

func TestHandlerOperation(t *testing.T) {
	suite.Run(t, new(testHandlerOperation))
}
