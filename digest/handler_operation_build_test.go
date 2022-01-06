//go:build mongodb
// +build mongodb

package digest

import (
	"bytes"
	"encoding/base64"
	"io"
	"testing"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
)

type testOperationBuildHandler struct {
	baseTestHandlers
}

func (t *testOperationBuildHandler) TestGET() {
	handlers := t.handlers(nil, DummyCache{})

	self, err := handlers.router.Get(HandlerPathOperationBuild).URL()
	t.NoError(err)

	w := t.requestOK(handlers, "GET", self.Path, nil)

	b, err := io.ReadAll(w.Result().Body)
	t.NoError(err)

	hal := t.loadHal(b)

	t.Equal(self.String(), hal.Links()["self"].Href())
}

func (t *testOperationBuildHandler) TestTemplate() {
	handlers := t.handlers(nil, DummyCache{})

	var factType string
	{
		factType = "create-accounts"
		self, err := handlers.router.Get(HandlerPathOperationBuildFactTemplate).URLPath("fact", factType)
		t.NoError(err)

		w := t.requestOK(handlers, "GET", self.Path, nil)

		b, err := io.ReadAll(w.Result().Body)
		t.NoError(err)

		hal := t.loadHal(b)

		t.Equal(self.String(), hal.Links()["self"].Href())
	}

	{
		factType = "key-updater"
		self, err := handlers.router.Get(HandlerPathOperationBuildFactTemplate).URLPath("fact", factType)
		t.NoError(err)

		w := t.requestOK(handlers, "GET", self.Path, nil)

		b, err := io.ReadAll(w.Result().Body)
		t.NoError(err)

		hal := t.loadHal(b)

		t.Equal(self.String(), hal.Links()["self"].Href())
	}

	{
		factType = "transfers"
		self, err := handlers.router.Get(HandlerPathOperationBuildFactTemplate).URLPath("fact", factType)
		t.NoError(err)

		w := t.requestOK(handlers, "GET", self.Path, nil)

		b, err := io.ReadAll(w.Result().Body)
		t.NoError(err)

		hal := t.loadHal(b)

		t.Equal(self.String(), hal.Links()["self"].Href())
	}
}

func (t *testOperationBuildHandler) TestPOSTFact() {
	handlers := t.handlers(nil, DummyCache{})

	factType := "transfers"
	self, err := handlers.router.Get(HandlerPathOperationBuildFactTemplate).URLPath("fact", factType)
	t.NoError(err)

	w := t.requestOK(handlers, "GET", self.Path, nil)

	var hal Hal
	{
		b, err := io.ReadAll(w.Result().Body)
		t.NoError(err)

		hal = t.loadHal(b)
	}

	t.Equal(self.String(), hal.Links()["self"].Href())

	newSender := currency.NewAddress("new-Mother")
	newReceiver := currency.NewAddress("new-Father")
	newBig := currency.NewBig(99)

	templateTokenEncoded := base64.StdEncoding.EncodeToString(templateToken)

	newToken := util.UUID().Bytes()
	newTokenEncoded := base64.StdEncoding.EncodeToString(newToken)

	b := bytes.ReplaceAll(
		hal.RawInterface(),
		[]byte(templateSender.String()),
		[]byte(newSender.String()),
	)
	b = bytes.ReplaceAll(
		b,
		[]byte(templateReceiver.String()),
		[]byte(newReceiver.String()),
	)
	b = bytes.ReplaceAll(
		b,
		[]byte(templateBig.String()),
		[]byte(newBig.String()),
	)
	b = bytes.ReplaceAll(
		b,
		[]byte(templateCurrencyID.String()),
		[]byte(t.cid.String()),
	)
	b = bytes.ReplaceAll(
		b,
		[]byte(templateTokenEncoded),
		[]byte(newTokenEncoded),
	)

	{ // NOTE at this time, fact has wrong hash
		hinter, err := t.JSONEnc.Decode(b)
		t.NoError(err)
		fact, ok := hinter.(currency.TransfersFact)
		t.True(ok)

		err = fact.IsValid(nil)
		t.Contains(err.Error(), "wrong Fact hash")

		t.Equal(newToken, fact.Token())
	}

	var opHal Hal
	{
		rw := t.requestOK(handlers, "POST", HandlerPathOperationBuildFact, b)
		b, err := io.ReadAll(rw.Result().Body)
		t.NoError(err)

		opHal = t.loadHal(b)
	}

	var op currency.Transfers
	var uf currency.TransfersFact
	{ // NOTE returned fact is valid
		hinter, err := t.JSONEnc.Decode(opHal.RawInterface())
		t.NoError(err)
		i, ok := hinter.(currency.Transfers)
		t.True(ok)
		op = i
		uf = op.Fact().(currency.TransfersFact)

		t.NoError(uf.IsValid(nil))
	}

	s, found := opHal.Extras()["signature_base"]
	t.True(found)
	t.NotEmpty(s)
	sigBase, err := base64.StdEncoding.DecodeString(s.(string))
	t.NoError(err)

	// expected sig base
	usigBase := base.NewBytesForFactSignature(uf, t.networkID)
	t.Equal(usigBase, sigBase)

	priv := key.NewBasePrivatekey()
	sig, err := priv.Sign(sigBase)
	t.NoError(err)

	b = bytes.ReplaceAll(
		opHal.RawInterface(),
		[]byte(templatePublickey.String()),
		[]byte(priv.Publickey().String()),
	)
	b = bytes.ReplaceAll(
		b,
		[]byte(base58.Encode(templateSignature)),
		[]byte(base58.Encode(sig)),
	)
	b = bytes.ReplaceAll(
		b,
		[]byte(`"memo":""`),
		[]byte(`"memo":"showme-memo"`),
	)

	var nopHal Hal
	{
		rw := t.requestOK(handlers, "POST", HandlerPathOperationBuildSign, b)

		b, err := io.ReadAll(rw.Result().Body)
		t.NoError(err)

		nopHal = t.loadHal(b)
	}

	var nop currency.Transfers
	{ // NOTE returned fact is valid
		hinter, err := t.JSONEnc.Decode(nopHal.RawInterface())
		t.NoError(err)
		i, ok := hinter.(currency.Transfers)
		t.True(ok)
		nop = i

		t.NoError(nop.IsValid(t.networkID))
	}
}

func TestOperationBuildHandler(t *testing.T) {
	suite.Run(t, new(testOperationBuildHandler))
}
