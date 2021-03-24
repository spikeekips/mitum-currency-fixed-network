// +build mongodb

package digest

import (
	"io"
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testHandlerBlock struct {
	baseTestHandlers
}

func (t *testHandlerBlock) TestByHeight() {
	st, _ := t.Database()
	handlers := t.handlers(st, DummyCache{})

	height := base.Height(33)
	self, err := handlers.router.Get(HandlerPathBlockByHeight).URLPath("height", height.String())
	t.NoError(err)

	w := t.requestOK(handlers, "GET", self.Path, nil)

	b, err := io.ReadAll(w.Result().Body)
	t.NoError(err)

	hal := t.loadHal(b)

	t.Equal(self.Path, hal.Links()["self"].Href())

	next, err := handlers.router.Get(HandlerPathBlockByHeight).URLPath("height", (height + 1).String())
	t.NoError(err)
	t.Equal(next.Path, hal.Links()["next"].Href())

	manifest, err := handlers.router.Get(HandlerPathManifestByHeight).URLPath("height", height.String())
	t.NoError(err)
	t.Equal(manifest.Path, hal.Links()["current-manifest"].Href())
}

func (t *testHandlerBlock) TestByHash() {
	st, _ := t.Database()
	handlers := t.handlers(st, DummyCache{})

	hash := valuehash.RandomSHA256()

	self, err := handlers.router.Get(HandlerPathBlockByHash).URLPath("hash", hash.String())
	t.NoError(err)

	w := t.requestOK(handlers, "GET", self.Path, nil)

	b, err := io.ReadAll(w.Result().Body)
	t.NoError(err)

	hal := t.loadHal(b)

	t.Equal(self.Path, hal.Links()["self"].Href())

	manifest, err := handlers.router.Get(HandlerPathManifestByHash).URLPath("hash", hash.String())
	t.NoError(err)
	t.Equal(manifest.Path, hal.Links()["manifest"].Href())
}

func TestHandlerBlock(t *testing.T) {
	suite.Run(t, new(testHandlerBlock))
}
