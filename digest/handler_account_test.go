//go:build mongodb
// +build mongodb

package digest

import (
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/stretchr/testify/suite"
)

type testHandlerAccount struct {
	baseTestHandlers
}

func (t *testHandlerAccount) TestAccount() {
	st, _ := t.Database()

	ac := t.newAccount()
	height := base.Height(33)

	am := currency.MustNewAmount(t.randomBig(), t.cid)

	va, _ := t.insertAccount(st, height, ac, am)

	handlers := t.handlers(st, DummyCache{})

	self, err := handlers.router.Get(HandlerPathAccount).URLPath("address", ac.Address().String())
	t.NoError(err)

	blockLink, err := handlers.router.Get(HandlerPathBlockByHeight).URLPath("height", va.Height().String())
	t.NoError(err)
	previousBlockLink, err := handlers.router.Get(HandlerPathBlockByHeight).URLPath("height", va.PreviousHeight().String())
	t.NoError(err)

	w := t.requestOK(handlers, "GET", self.Path, nil)

	b, err := io.ReadAll(w.Result().Body)
	t.NoError(err)

	hal := t.loadHal(b)

	t.Equal(self.String(), hal.Links()["self"].Href())
	t.Equal(blockLink.Path, hal.Links()["block"].Href())
	t.Equal(previousBlockLink.Path, hal.Links()["previous_block"].Href())

	hinter, err := t.JSONEnc.Decode(hal.RawInterface())
	t.NoError(err)
	uva, ok := hinter.(AccountValue)
	t.True(ok)

	t.compareAccountValue(va, uva)
}

func (t *testHandlerAccount) TestAccountNotFound() {
	st, _ := t.Database()

	handlers := t.handlers(st, DummyCache{})

	unknown := currency.NewAddress(util.UUID().String())
	t.NoError(unknown.IsValid(nil))

	self, err := handlers.router.Get(HandlerPathAccount).URLPath("address", unknown.String())
	t.NoError(err)

	w := t.request404(handlers, "GET", self.Path, nil)

	b, err := io.ReadAll(w.Result().Body)
	t.NoError(err)

	var problem Problem
	t.NoError(jsonenc.Unmarshal(b, &problem))
	t.Contains(problem.Error(), "not found")
}

func (t *testHandlerAccount) TestAccountOperations() {
	st, _ := t.Database()

	sender := currency.MustAddress(util.UUID().String())

	var offsets []string
	offsetByHashes := map[string]string{}
	hashesByOffset := map[string]string{}

	for i := 0; i < 10; i++ {
		height := base.Height(i % 3)
		index := uint64(i)
		tf := t.newTransfer(sender, currency.MustAddress(util.UUID().String()))
		doc, err := NewOperationDoc(tf, t.BSONEnc, height, localtime.UTCNow(), true, nil, index)
		t.NoError(err)
		_ = t.insertDoc(st, defaultColNameOperation, doc)

		fh := tf.Fact().Hash().String()
		offset := buildOffset(height, index)
		offsets = append(offsets, offset)
		offsetByHashes[fh] = offset
		hashesByOffset[offset] = fh
	}

	var hashes []string
	sort.Strings(offsets)
	for _, o := range offsets {
		hashes = append(hashes, hashesByOffset[o])
	}

	var limit int64 = 3
	handlers := t.handlers(st, DummyCache{})
	_ = handlers.SetLimiter(func(string) int64 {
		return limit
	})

	self, err := handlers.router.Get(HandlerPathAccountOperations).URLPath("address", sender.String())
	t.NoError(err)

	next, err := handlers.router.Get(HandlerPathAccountOperations).URLPath("address", sender.String())
	t.NoError(err)
	next.RawQuery = stringOffsetQuery(offsetByHashes[hashes[limit-1]])

	w := t.requestOK(handlers, "GET", self.Path, nil)

	b, err := io.ReadAll(w.Result().Body)
	t.NoError(err)

	hal := t.loadHal(b)

	// NOTE check self link
	t.Equal(self.String(), hal.Links()["self"].Href())

	// NOTE check next link
	t.Equal(next.String(), hal.Links()["next"].Href())

	var em []BaseHal
	t.NoError(jsonenc.Unmarshal(hal.RawInterface(), &em))
	t.Equal(int(limit), len(em))
}

func (t *testHandlerAccount) getHashes(handlers *Handlers, limit int, self *url.URL) []string {
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

func (t *testHandlerAccount) TestAccountOperationsPaging() {
	st, _ := t.Database()

	sender := currency.MustAddress(util.UUID().String())
	var offsets []string
	hashesByOffset := map[string]string{}

	for i := 0; i < 10; i++ {
		height := base.Height(i % 3)
		index := uint64(i)
		tf := t.newTransfer(sender, currency.MustAddress(util.UUID().String()))
		doc, err := NewOperationDoc(tf, t.BSONEnc, height, localtime.UTCNow(), true, nil, index)
		t.NoError(err)
		_ = t.insertDoc(st, defaultColNameOperation, doc)

		fh := tf.Fact().Hash().String()

		offset := buildOffset(height, index)
		offsets = append(offsets, offset)
		hashesByOffset[offset] = fh
	}

	var limit int64 = 3
	handlers := t.handlers(st, DummyCache{})
	_ = handlers.SetLimiter(func(string) int64 {
		return limit
	})

	{ // no reverse
		var hashes []string
		sort.Strings(offsets)
		for _, o := range offsets {
			hashes = append(hashes, hashesByOffset[o])
		}

		reverse := false
		offset := ""

		self, err := handlers.router.Get(HandlerPathAccountOperations).URLPath("address", sender.String())
		t.NoError(err)
		self.RawQuery = fmt.Sprintf("%s&%s", stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))

		uhashes := t.getHashes(handlers, int(limit), self)
		t.Equal(hashes, uhashes)
	}

	{ // reverse
		var hashes []string
		sort.Sort(sort.Reverse(sort.StringSlice(offsets)))

		for _, o := range offsets {
			hashes = append(hashes, hashesByOffset[o])
		}

		reverse := true
		offset := ""

		self, err := handlers.router.Get(HandlerPathAccountOperations).URLPath("address", sender.String())
		t.NoError(err)
		self.RawQuery = fmt.Sprintf("%s&%s", stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))

		uhashes := t.getHashes(handlers, int(limit), self)
		t.Equal(hashes, uhashes)
	}
}

func (t *testHandlerAccount) TestAccountOperationsPagingOverOffset() {
	st, _ := t.Database()

	sender := currency.MustAddress(util.UUID().String())

	var hashes []string
	for i := 0; i < 10; i++ {
		height := base.Height(i % 3)
		tf := t.newTransfer(sender, currency.MustAddress(util.UUID().String()))
		doc, err := NewOperationDoc(tf, t.BSONEnc, height, localtime.UTCNow(), true, nil, uint64(i))
		t.NoError(err)
		_ = t.insertDoc(st, defaultColNameOperation, doc)

		hashes = append(hashes, tf.Fact().Hash().String())
	}

	sort.Sort(sort.Reverse(sort.StringSlice(hashes)))

	var limit int64 = 3
	handlers := t.handlers(st, DummyCache{})
	_ = handlers.SetLimiter(func(string) int64 {
		return limit
	})

	offset := buildOffset(base.Height(9), uint64(20))

	self, err := handlers.router.Get(HandlerPathAccountOperations).URLPath("address", sender.String())
	self.RawQuery = fmt.Sprintf("%s&", stringOffsetQuery(offset))
	t.NoError(err)

	w := t.request404(handlers, "GET", self.String(), nil)

	b, err := io.ReadAll(w.Result().Body)
	t.NoError(err)

	var problem Problem
	t.NoError(jsonenc.Unmarshal(b, &problem))
	t.Contains(problem.Error(), "operations not found")
}

func (t *testHandlerAccount) TestAccountOperationsReverseCache() {
	st, _ := t.Database()

	insert := func(height base.Height, sender base.Address, l int) ([]string, map[string]string) {
		var offsets []string
		hashesByOffset := map[string]string{}

		for i := uint64(0); i < uint64(l); i++ {
			tf := t.newTransfer(sender, currency.MustAddress(util.UUID().String()))
			doc, err := NewOperationDoc(tf, t.BSONEnc, height, localtime.UTCNow(), true, nil, i)
			t.NoError(err)
			_ = t.insertDoc(st, defaultColNameOperation, doc)

			fh := tf.Fact().Hash().String()

			offset := buildOffset(height, i)
			offsets = append(offsets, offset)
			hashesByOffset[offset] = fh
		}

		return offsets, hashesByOffset
	}

	var limit int64 = 3
	handlers := t.handlers(st, NewLocalMemCache(1000, time.Minute))
	_ = handlers.SetLimiter(func(string) int64 {
		return limit
	})
	handlers.expireNotFilled = time.Second * 2

	sender := currency.MustAddress(util.UUID().String())
	offsets, hashesByOffset := insert(base.Height(3), sender, 10)

	{ // reverse
		var hashes []string
		sort.Sort(sort.Reverse(sort.StringSlice(offsets)))

		for _, o := range offsets {
			hashes = append(hashes, hashesByOffset[o])
		}

		reverse := true
		offset := ""

		self, err := handlers.router.Get(HandlerPathAccountOperations).URLPath("address", sender.String())
		t.NoError(err)
		self.RawQuery = fmt.Sprintf("%s&%s", stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))

		uhashes := t.getHashes(handlers, int(limit), self)

		t.Equal(hashes, uhashes)
	}

	t.T().Log("insert more")
	{
		o, h := insert(base.Height(4), sender, 1)
		for i := range o {
			offsets = append(offsets, o[i])

			for i := range h {
				hashesByOffset[i] = h[i]
			}
		}
	}

	<-time.After(handlers.expireNotFilled + time.Millisecond) // wait empty offset expire

	{ // reverse again
		sort.Sort(sort.Reverse(sort.StringSlice(offsets)))

		var hashes []string
		for _, o := range offsets {
			hashes = append(hashes, hashesByOffset[o])
		}

		reverse := true
		offset := ""

		self, err := handlers.router.Get(HandlerPathAccountOperations).URLPath("address", sender.String())
		t.NoError(err)
		self.RawQuery = fmt.Sprintf("%s&%s", stringOffsetQuery(offset), stringBoolQuery("reverse", reverse))

		uhashes := t.getHashes(handlers, int(limit), self)

		t.Equal(hashes, uhashes)
	}
}

func (t *testHandlerAccount) TestAccounts() {
	st, _ := t.Database()

	priv := key.NewBasePrivatekey()
	k, err := currency.NewBaseAccountKey(priv.Publickey(), 100)
	t.NoError(err)

	var topHeight base.Height

	var olds []AccountValue
	for h := base.Height(3); h < 6; h++ {
		if h > topHeight {
			topHeight = h
		}

		var vas []AccountValue
		for i := 0; i < 3; i++ {
			ac := t.newAccount()
			keys, err := currency.NewBaseAccountKeys([]currency.AccountKey{k}, 100)
			t.NoError(err)
			ac, err = ac.SetKeys(keys)
			t.NoError(err)

			am := currency.MustNewAmount(t.randomBig(), t.cid)

			va, _ := t.insertAccount(st, h, ac, am)
			vas = append(vas, va)
		}

		olds = append(olds, vas...)
	}

	oldsHigher := make([]AccountValue, len(olds))
	for i := range olds {
		ova := olds[i]

		va, _ := t.insertAccount(st, ova.height+1, ova.ac, ova.balance[0])

		oldsHigher[i] = va
	}

	sames := make([]AccountValue, len(oldsHigher)-2)

	newpriv := key.NewBasePrivatekey()
	newkey, err := currency.NewBaseAccountKey(newpriv.Publickey(), 100)
	t.NoError(err)

	for i := range oldsHigher {
		ova := oldsHigher[i]

		if i > len(oldsHigher)-3 {
			// NOTE remove pubkey from account
			keys, err := currency.NewBaseAccountKeys([]currency.AccountKey{newkey}, 100)
			t.NoError(err)
			ac, err := ova.ac.SetKeys(keys)
			t.NoError(err)

			_, _ = t.insertAccount(st, ova.height+1, ac, ova.balance[0])

			if ova.height+1 > topHeight {
				topHeight = ova.height + 1
			}

			continue
		}

		sames[i] = ova
	}

	sort.Slice(sames, func(i, j int) bool {
		return strings.Compare(sames[i].Account().Address().String(), sames[j].Account().Address().String()) < 0
	})

	for i := 0; i < 3; i++ {
		ac := t.newAccount()

		am := currency.MustNewAmount(t.randomBig(), t.cid)

		_, _ = t.insertAccount(st, base.Height(4), ac, am)
	}

	var limit int64 = 4
	handlers := t.handlers(st, NewLocalMemCache(1000, time.Minute))
	_ = handlers.SetLimiter(func(string) int64 {
		return limit
	})
	handlers.expireNotFilled = 0

	queries := url.Values{}
	queries.Set("publickey", priv.Publickey().String())

	self, err := handlers.router.Get(HandlerPathAccounts).URL()
	self.RawQuery = queries.Encode()
	t.NoError(err)

	{
		items := t.getItems(handlers, int(limit), self, func(b []byte) (interface{}, error) {
			return t.JSONEnc.Decode(b)
		})
		t.Equal(len(sames), len(items))

		for i := range items {
			a := sames[i].Account()
			b := items[i].(AccountValue).Account()

			t.compareAccount(a, b)
		}
	}

	{ // again; check cached
		items := t.getItems(handlers, int(limit), self, func(b []byte) (interface{}, error) {
			return t.JSONEnc.Decode(b)
		})
		t.Equal(len(sames), len(items))

		for i := range items {
			a := sames[i].Account()
			b := items[i].(AccountValue).Account()

			t.compareAccount(a, b)
		}
	}

	topHeight++
	updated := make([]AccountValue, len(sames)-2)
	for i := range sames {
		ova := sames[i]

		if i > len(sames)-3 {
			// NOTE remove pubkey from account
			keys, err := currency.NewBaseAccountKeys([]currency.AccountKey{newkey}, 100)
			t.NoError(err)
			ac, err := ova.ac.SetKeys(keys)
			t.NoError(err)

			_, _ = t.insertAccount(st, topHeight, ac, ova.balance[0])

			continue
		}

		updated[i] = ova
	}

	{ // again; check updated
		items := t.getItems(handlers, int(limit), self, func(b []byte) (interface{}, error) {
			return t.JSONEnc.Decode(b)
		})
		t.Equal(len(updated), len(items))

		for i := range items {
			a := updated[i].Account()
			b := items[i].(AccountValue).Account()

			t.compareAccount(a, b)
		}
	}
}

func TestHandlerAccount(t *testing.T) {
	suite.Run(t, new(testHandlerAccount))
}
