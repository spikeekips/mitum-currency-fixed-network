//go:build mongodb
// +build mongodb

package digest

import (
	"fmt"
	"testing"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/stretchr/testify/suite"
)

type testHandlerSend struct {
	baseTestHandlers
}

func (t *testHandlerSend) TestEmptyRemoteNodes() {
	st, _ := t.Database()
	handlers := t.handlers(st, DummyCache{})

	self, err := handlers.router.Get(HandlerPathSend).URL()
	t.NoError(err)

	_ = t.request405(handlers, "GET", self.String(), nil)

	op := t.newTransfer(currency.MustAddress(util.UUID().String()), currency.MustAddress(util.UUID().String()))

	b, err := jsonenc.Marshal(op)
	t.NoError(err)

	_, problem := t.request500(handlers, "POST", self.String(), b)

	t.Contains(problem.Error(), "not supported")
}

func (t *testHandlerSend) TestSend() {
	st, _ := t.Database()
	handlers := t.handlers(st, DummyCache{})

	var sent operation.Operation
	handlers.SetSend(func(sl interface{}) (seal.Seal, error) {
		sent = sl.(operation.Operation)

		return nil, nil
	})

	self, err := handlers.router.Get(HandlerPathSend).URL()
	t.NoError(err)

	op := t.newTransfer(currency.MustAddress(util.UUID().String()), currency.MustAddress(util.UUID().String()))

	b, err := jsonenc.Marshal(op)
	t.NoError(err)

	_ = t.requestOK(handlers, "POST", self.String(), b)

	t.True(op.Hash().Equal(sent.Hash()))
	t.True(op.Fact().Hash().Equal(sent.Fact().Hash()))
}

func (t *testHandlerSend) TestSendFailed() {
	st, _ := t.Database()
	handlers := t.handlers(st, DummyCache{})

	handlers.SetSend(func(sl interface{}) (seal.Seal, error) {
		return nil, fmt.Errorf("findme")
	})

	self, err := handlers.router.Get(HandlerPathSend).URL()
	t.NoError(err)

	op := t.newTransfer(currency.MustAddress(util.UUID().String()), currency.MustAddress(util.UUID().String()))

	b, err := jsonenc.Marshal(op)
	t.NoError(err)

	_, problem := t.request400(handlers, "POST", self.String(), b)
	t.Equal("findme", problem.Error())
}

func (t *testHandlerSend) TestSendOperations() {
	st, _ := t.Database()
	handlers := t.handlers(st, DummyCache{})

	var sent operation.Seal
	handlers.SetSend(func(sl interface{}) (seal.Seal, error) {
		sent = sl.(operation.Seal)

		return nil, nil
	})

	self, err := handlers.router.Get(HandlerPathSend).URL()
	t.NoError(err)

	var ops []operation.Operation
	{
		op := t.newTransfer(currency.MustAddress(util.UUID().String()), currency.MustAddress(util.UUID().String()))
		ops = append(ops, op)

		op = t.newTransfer(currency.MustAddress(util.UUID().String()), currency.MustAddress(util.UUID().String()))
		ops = append(ops, op)
	}

	b, err := jsonenc.Marshal(ops)
	t.NoError(err)

	_ = t.requestOK(handlers, "POST", self.String(), b)

	t.Equal(len(ops), len(sent.Operations()))

	for i := range ops {
		a := ops[i]
		b := sent.Operations()[i]

		t.True(a.Hash().Equal(b.Hash()))
		t.True(a.Fact().Hash().Equal(b.Fact().Hash()))
	}
}

func TestHandlerSend(t *testing.T) {
	suite.Run(t, new(testHandlerSend))
}
