//go:build mongodb
// +build mongodb

package digest

import (
	"fmt"
	"io"
	"net/url"
	"testing"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util/encoder"
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
	t.True(localtime.Equal(ma.ConfirmedAt(), mb.ConfirmedAt()))
}

func (t *testHandlerManifest) TestByHeight() {
	st, mst := t.Database()
	handlers := t.handlers(st, DummyCache{})

	height := base.Height(33)

	blk := t.newBlock(height, mst)

	self, err := handlers.router.Get(HandlerPathManifestByHeight).URLPath("height", height.String())
	t.NoError(err)

	w := t.requestOK(handlers, "GET", self.Path, nil)

	b, err := io.ReadAll(w.Result().Body)
	t.NoError(err)

	hal := t.loadHal(b)

	t.Equal(self.Path, hal.Links()["self"].Href())

	next, err := handlers.router.Get(HandlerPathManifestByHeight).URLPath("height", (height + 1).String())
	t.NoError(err)
	t.Equal(next.Path, hal.Links()["next"].Href())

	hinter, err := t.JSONEnc.Decode(hal.RawInterface())
	t.NoError(err)

	t.compareManifest(blk, hinter)
}

func (t *testHandlerManifest) TestByHeightNotFound() {
	st, _ := t.Database()
	handlers := t.handlers(st, DummyCache{})

	height := base.Height(33)

	self, err := handlers.router.Get(HandlerPathManifestByHeight).URLPath("height", height.String())
	t.NoError(err)

	w := t.request404(handlers, "GET", self.Path, nil)

	b, err := io.ReadAll(w.Result().Body)
	t.NoError(err)

	var problem Problem
	t.NoError(jsonenc.Unmarshal(b, &problem))
	t.Contains(problem.Error(), "manifest not found")
}

func (t *testHandlerManifest) TestByHash() {
	st, mst := t.Database()
	handlers := t.handlers(st, DummyCache{})

	height := base.Height(33)

	blk := t.newBlock(height, mst)

	self, err := handlers.router.Get(HandlerPathManifestByHeight).URLPath("height", blk.Height().String())
	t.NoError(err)
	nonself, err := handlers.router.Get(HandlerPathManifestByHash).URLPath("hash", blk.Hash().String())
	t.NoError(err)

	w := t.requestOK(handlers, "GET", nonself.Path, nil)

	b, err := io.ReadAll(w.Result().Body)
	t.NoError(err)

	hal := t.loadHal(b)

	t.Equal(self.Path, hal.Links()["self"].Href())

	next, err := handlers.router.Get(HandlerPathManifestByHeight).URLPath("height", (blk.Height() + 1).String())
	t.NoError(err)
	t.Equal(next.Path, hal.Links()["next"].Href())

	hinter, err := t.JSONEnc.Decode(hal.RawInterface())
	t.NoError(err)

	t.compareManifest(blk, hinter)
}

func (t *testHandlerManifest) TestByHashNotFound() {
	st, _ := t.Database()
	handlers := t.handlers(st, DummyCache{})

	self, err := handlers.router.Get(HandlerPathManifestByHash).URLPath("hash", valuehash.RandomSHA256().String())
	t.NoError(err)

	w := t.request404(handlers, "GET", self.Path, nil)

	b, err := io.ReadAll(w.Result().Body)
	t.NoError(err)

	var problem Problem
	t.NoError(jsonenc.Unmarshal(b, &problem))
	t.Contains(problem.Error(), "manifest not found")
}

func (t *testHandlerManifest) getManifests(handlers *Handlers, limit int, self *url.URL) []block.Manifest {
	l := t.getItems(handlers, limit, self, func(b []byte) (interface{}, error) {
		var m block.Manifest
		err := encoder.Decode(b, t.JSONEnc, &m)

		return m, err
	})

	ms := make([]block.Manifest, len(l))
	for i := range l {
		ms[i] = l[i].(block.Manifest)
	}

	return ms
}

func (t *testHandlerManifest) TestManifests() {
	st, mst := t.Database()

	var baseheight int64 = 33

	var blocks []block.Manifest
	for i := int64(0); i < 7; i++ {
		blk := t.newBlock(base.Height(baseheight+i), mst)
		blocks = append(blocks, blk.Manifest())
	}

	var limit int64 = 3
	handlers := t.handlers(st, DummyCache{})
	_ = handlers.SetLimiter(func(string) int64 {
		return limit
	})

	{ // no reverse
		reverse := false
		offset := ""

		self, err := handlers.router.Get(HandlerPathManifests).URL()
		t.NoError(err)
		self.RawQuery = fmt.Sprintf("%s&%s", stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))

		ublocks := t.getManifests(handlers, int(limit), self)
		t.Equal(len(blocks), len(ublocks))

		for i := range blocks {
			t.compareManifest(blocks[i], ublocks[i])
		}
	}

	{ // reverse
		reverse := true
		offset := ""

		self, err := handlers.router.Get(HandlerPathManifests).URL()
		t.NoError(err)
		self.RawQuery = fmt.Sprintf("%s&%s", stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))

		ublocks := t.getManifests(handlers, int(limit), self)
		t.Equal(len(blocks), len(ublocks))

		for i := range blocks {
			t.compareManifest(blocks[i], ublocks[len(blocks)-i-1])
		}
	}

	{ // offset
		reverse := false
		offset := blocks[3].Height().String()

		self, err := handlers.router.Get(HandlerPathManifests).URL()
		t.NoError(err)
		self.RawQuery = fmt.Sprintf("%s&%s", stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))

		ublocks := t.getManifests(handlers, int(limit), self)
		t.Equal(len(blocks[4:]), len(ublocks))

		for i, m := range blocks[4:] {
			t.compareManifest(m, ublocks[i])
		}
	}
}

func (t *testHandlerManifest) TestManifestsCache() {
	st, mst := t.Database()

	var baseheight int64 = 33

	var blocks []block.Manifest
	for i := int64(0); i < 7; i++ {
		blk := t.newBlock(base.Height(baseheight+i), mst)
		blocks = append(blocks, blk.Manifest())
	}

	var limit int64 = 3
	handlers := t.handlers(st, NewLocalMemCache(1000, time.Minute))
	_ = handlers.SetLimiter(func(string) int64 {
		return limit
	})
	handlers.expireNotFilled = time.Second * 2

	reverse := true
	offset := ""

	self, err := handlers.router.Get(HandlerPathManifests).URL()
	t.NoError(err)
	self.RawQuery = fmt.Sprintf("%s&%s", stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))

	ublocks := t.getManifests(handlers, int(limit), self)
	t.Equal(len(blocks), len(ublocks))

	for i := range blocks {
		t.compareManifest(blocks[i], ublocks[len(blocks)-i-1])
	}

	t.T().Log("insert more")

	baseheight = blocks[len(blocks)-1].Height().Int64() + 1
	for i := int64(0); i < 7; i++ {
		blk := t.newBlock(base.Height(baseheight+i), mst)
		blocks = append(blocks, blk.Manifest())
	}

	<-time.After(handlers.expireNotFilled + time.Millisecond) // wait empty offset expire
	offset = ""

	self, err = handlers.router.Get(HandlerPathManifests).URL()
	t.NoError(err)
	self.RawQuery = fmt.Sprintf("%s&%s", stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))

	ublocks = t.getManifests(handlers, int(limit), self)
	t.Equal(len(blocks), len(ublocks))
	for i := range blocks {
		t.compareManifest(blocks[i], ublocks[len(blocks)-i-1])
	}
}

func TestHandlerManifest(t *testing.T) {
	suite.Run(t, new(testHandlerManifest))
}
