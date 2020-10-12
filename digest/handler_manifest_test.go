// +build mongodb

package digest

import (
	"io/ioutil"
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testHandlerManifest struct {
	baseTestHandlers
}

func (t *testHandlerManifest) compareManifest(a, b interface{}) {
	ma := a.(block.Manifest)
	mb := b.(block.Manifest)

	t.Equal(ma.Height(), mb.Height())
	t.Equal(ma.Round(), mb.Round())
	t.True(ma.Hash().Equal(mb.Hash()))
	t.True(ma.Proposal().Equal(mb.Proposal()))
	t.True(ma.OperationsHash().Equal(mb.OperationsHash()))
	t.True(ma.StatesHash().Equal(mb.StatesHash()))
	t.Equal(localtime.Normalize(ma.ConfirmedAt()), localtime.Normalize(mb.ConfirmedAt()))
}

func (t *testHandlerManifest) TestByHeight() {
	st, mst := t.Storage()
	handlers := t.handlers(st, DummyCache{})

	height := base.Height(33)

	blk := t.newBlock(height, mst)

	self, err := handlers.router.Get(HandlerPathManifestByHeight).URLPath("height", height.String())
	t.NoError(err)

	w := t.requestOK(handlers, "GET", self.Path, nil)

	b, err := ioutil.ReadAll(w.Result().Body)
	t.NoError(err)

	hal := t.loadHal(b)

	t.Equal(self.Path, hal.Links()["self"].Href())

	next, err := handlers.router.Get(HandlerPathManifestByHeight).URLPath("height", (height + 1).String())
	t.NoError(err)
	t.Equal(next.Path, hal.Links()["next"].Href())

	hinter, err := t.JSONEnc.DecodeByHint(hal.RawInterface())
	t.NoError(err)

	t.compareManifest(blk, hinter)
}

func (t *testHandlerManifest) TestByHeightNotFound() {
	st, _ := t.Storage()
	handlers := t.handlers(st, DummyCache{})

	height := base.Height(33)

	self, err := handlers.router.Get(HandlerPathManifestByHeight).URLPath("height", height.String())
	t.NoError(err)

	w := t.request404(handlers, "GET", self.Path, nil)

	b, err := ioutil.ReadAll(w.Result().Body)
	t.NoError(err)

	var problem Problem
	t.NoError(jsonenc.Unmarshal(b, &problem))
	t.Contains(problem.Error(), "manifest not found")
}

func (t *testHandlerManifest) TestByHash() {
	st, mst := t.Storage()
	handlers := t.handlers(st, DummyCache{})

	height := base.Height(33)

	blk := t.newBlock(height, mst)

	self, err := handlers.router.Get(HandlerPathManifestByHeight).URLPath("height", blk.Height().String())
	t.NoError(err)
	nonself, err := handlers.router.Get(HandlerPathManifestByHash).URLPath("hash", blk.Hash().String())
	t.NoError(err)

	w := t.requestOK(handlers, "GET", nonself.Path, nil)

	b, err := ioutil.ReadAll(w.Result().Body)
	t.NoError(err)

	hal := t.loadHal(b)

	t.Equal(self.Path, hal.Links()["self"].Href())

	next, err := handlers.router.Get(HandlerPathManifestByHeight).URLPath("height", (blk.Height() + 1).String())
	t.NoError(err)
	t.Equal(next.Path, hal.Links()["next"].Href())

	hinter, err := t.JSONEnc.DecodeByHint(hal.RawInterface())
	t.NoError(err)

	t.compareManifest(blk, hinter)
}

func (t *testHandlerManifest) TestByHashNotFound() {
	st, _ := t.Storage()
	handlers := t.handlers(st, DummyCache{})

	self, err := handlers.router.Get(HandlerPathManifestByHash).URLPath("hash", valuehash.RandomSHA256().String())
	t.NoError(err)

	w := t.request404(handlers, "GET", self.Path, nil)

	b, err := ioutil.ReadAll(w.Result().Body)
	t.NoError(err)

	var problem Problem
	t.NoError(jsonenc.Unmarshal(b, &problem))
	t.Contains(problem.Error(), "manifest not found")
}

func TestHandlerManifest(t *testing.T) {
	suite.Run(t, new(testHandlerManifest))
}
