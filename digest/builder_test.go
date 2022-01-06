//go:build mongodb
// +build mongodb

package digest

import (
	"bytes"
	"encoding/base64"
	"testing"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/stretchr/testify/suite"
)

type testBuilder struct {
	baseTest
}

func (t *testBuilder) decodeHal(b []byte) Hal {
	hinter, err := t.JSONEnc.Decode(b)
	if err != nil {
		panic(err)
	}

	hal, ok := hinter.(Hal)
	if !ok {
		panic("not Hal")
	}

	hinter, err = t.JSONEnc.Decode(hal.RawInterface())
	if err != nil {
		panic(err)
	}

	return hal.SetInterface(hinter)
}

func (t *testBuilder) TestUnknownFactTemplate() {
	bl := NewBuilder(t.JSONEnc, t.networkID)

	_, err := bl.FactTemplate(currency.CreateAccountsHinter.Hint())
	t.NoError(err)

	_, err = bl.FactTemplate(currency.KeyUpdaterHinter.Hint())
	t.NoError(err)

	_, err = bl.FactTemplate(currency.TransfersHinter.Hint())
	t.NoError(err)

	_, err = bl.FactTemplate(currency.GenesisCurrenciesHinter.Hint())
	t.Contains(err.Error(), "unknown operation")
}

func (t *testBuilder) TestFactTemplateCreateAccounts() {
	bl := NewBuilder(t.JSONEnc, t.networkID)

	hal, err := bl.FactTemplate(currency.CreateAccountsHinter.Hint())
	t.NoError(err)
	t.NotEmpty(hal.Extras())

	b, err := t.JSONEnc.Marshal(hal)
	t.NoError(err)
	uhal := t.decodeHal(b)

	t.IsType(currency.CreateAccountsFact{}, uhal.Interface())
}

func (t *testBuilder) TestFactTemplateKeyUpdater() {
	bl := NewBuilder(t.JSONEnc, t.networkID)

	hal, err := bl.FactTemplate(currency.KeyUpdaterHinter.Hint())
	t.NoError(err)
	t.NotEmpty(hal.Extras())

	b, err := t.JSONEnc.Marshal(hal)
	t.NoError(err)
	uhal := t.decodeHal(b)

	t.IsType(currency.KeyUpdaterFact{}, uhal.Interface())
}

func (t *testBuilder) TestFactTemplateTransfers() {
	bl := NewBuilder(t.JSONEnc, t.networkID)

	hal, err := bl.FactTemplate(currency.TransfersHinter.Hint())
	t.NoError(err)
	t.NotEmpty(hal.Extras())

	b, err := t.JSONEnc.Marshal(hal)
	t.NoError(err)
	uhal := t.decodeHal(b)

	t.IsType(currency.TransfersFact{}, uhal.Interface())
}

func (t *testBuilder) TestFactTemplateCurrencyRegister() {
	bl := NewBuilder(t.JSONEnc, t.networkID)

	hal, err := bl.FactTemplate(currency.CurrencyRegisterHinter.Hint())
	t.NoError(err)
	t.NotEmpty(hal.Extras())

	b, err := t.JSONEnc.Marshal(hal)
	t.NoError(err)
	uhal := t.decodeHal(b)

	t.IsType(currency.CurrencyRegisterFact{}, uhal.Interface())
}

func (t *testBuilder) TestFactTemplateCurrencyPolicyUpdater() {
	bl := NewBuilder(t.JSONEnc, t.networkID)

	hal, err := bl.FactTemplate(currency.CurrencyPolicyUpdaterHinter.Hint())
	t.NoError(err)
	t.NotEmpty(hal.Extras())

	b, err := t.JSONEnc.Marshal(hal)
	t.NoError(err)
	uhal := t.decodeHal(b)

	t.IsType(currency.CurrencyPolicyUpdaterFact{}, uhal.Interface())
}

func (t *testBuilder) TestBuildFactCreateAccounts() {
	bl := NewBuilder(t.JSONEnc, t.networkID)

	hal, err := bl.FactTemplate(currency.CreateAccountsHinter.Hint())
	t.NoError(err)

	b, err := t.JSONEnc.Marshal(hal)
	t.NoError(err)
	rhal := t.decodeHal(b)

	templateTokenEncoded := base64.StdEncoding.EncodeToString(templateToken)

	newPub := key.NewBasePrivatekey().Publickey()
	newSender := currency.NewAddress("new-mother")
	newBig := currency.NewBig(99)
	newToken := util.UUID().Bytes()
	newTokenEncoded := base64.StdEncoding.EncodeToString(newToken)
	newCurrencyID := currency.CurrencyID("XXX")

	b = bytes.ReplaceAll(rhal.RawInterface(), []byte(templateSender.String()), []byte(newSender.String()))
	b = bytes.ReplaceAll(b, []byte(templatePublickey.String()), []byte(newPub.String()))
	b = bytes.ReplaceAll(b, []byte(templateBig.String()), []byte(newBig.String()))
	b = bytes.ReplaceAll(b, []byte(templateTokenEncoded), []byte(newTokenEncoded))
	b = bytes.ReplaceAll(b, []byte(templateCurrencyID), newCurrencyID.Bytes())

	uhal, err := bl.BuildFact(b)
	t.NoError(err)

	uop, ok := uhal.Interface().(currency.CreateAccounts)
	t.True(ok)
	err = uop.IsValid(nil)
	t.Contains(err.Error(), "malformed signature")

	ufact := uop.Fact().(currency.CreateAccountsFact)

	t.True(ufact.Sender().Equal(newSender))
	t.Equal(ufact.Token(), newToken)
	t.Equal(1, len(ufact.Items()[0].Amounts()))
	t.Equal(newBig, ufact.Items()[0].Amounts()[0].Big())
	t.Equal(newCurrencyID, ufact.Items()[0].Amounts()[0].Currency())

	_, same := ufact.Items()[0].Keys().Key(newPub)
	t.True(same)

	sb, found := uhal.Extras()["signature_base"]
	t.True(found)

	_ = t.buildOperation(uop, sb.([]byte))
}

func (t *testBuilder) TestBuildFactKeyUpdater() {
	bl := NewBuilder(t.JSONEnc, t.networkID)

	hal, err := bl.FactTemplate(currency.KeyUpdaterHinter.Hint())
	t.NoError(err)

	b, err := t.JSONEnc.Marshal(hal)
	t.NoError(err)
	rhal := t.decodeHal(b)

	templateTokenEncoded := base64.StdEncoding.EncodeToString(templateToken)

	newPub := key.NewBasePrivatekey().Publickey()
	newSender := currency.NewAddress("new-mother")
	newToken := util.UUID().Bytes()
	newTokenEncoded := base64.StdEncoding.EncodeToString(newToken)
	newCurrencyID := currency.CurrencyID("XXX")

	b = bytes.ReplaceAll(rhal.RawInterface(), []byte(templateSender.String()), []byte(newSender.String()))
	b = bytes.ReplaceAll(b, []byte(templatePublickey.String()), []byte(newPub.String()))
	b = bytes.ReplaceAll(b, []byte(templateTokenEncoded), []byte(newTokenEncoded))
	b = bytes.ReplaceAll(b, []byte(templateCurrencyID), newCurrencyID.Bytes())

	uhal, err := bl.BuildFact(b)
	t.NoError(err)

	uop, ok := uhal.Interface().(currency.KeyUpdater)
	t.True(ok)
	err = uop.IsValid(nil)
	t.Contains(err.Error(), "malformed signature")

	ufact := uop.Fact().(currency.KeyUpdaterFact)

	t.True(ufact.Target().Equal(newSender))

	_, same := ufact.Keys().Key(newPub)
	t.True(same)
	t.Equal(newCurrencyID, ufact.Currency())

	sb, found := uhal.Extras()["signature_base"]
	t.True(found)

	_ = t.buildOperation(uop, sb.([]byte))
}

func (t *testBuilder) TestBuildFactTransfers() {
	bl := NewBuilder(t.JSONEnc, t.networkID)

	hal, err := bl.FactTemplate(currency.TransfersHinter.Hint())
	t.NoError(err)

	b, err := t.JSONEnc.Marshal(hal)
	t.NoError(err)
	rhal := t.decodeHal(b)

	templateTokenEncoded := base64.StdEncoding.EncodeToString(templateToken)

	newSender := currency.NewAddress("new-mother")
	newReceiver := currency.NewAddress("new-father")
	newBig := currency.NewBig(99)
	newToken := util.UUID().Bytes()
	newTokenEncoded := base64.StdEncoding.EncodeToString(newToken)
	newCurrencyID := currency.CurrencyID("XXX")

	b = bytes.ReplaceAll(rhal.RawInterface(), []byte(templateSender.String()), []byte(newSender.String()))
	b = bytes.ReplaceAll(b, []byte(templateReceiver.String()), []byte(newReceiver.String()))
	b = bytes.ReplaceAll(b, []byte(templateBig.String()), []byte(newBig.String()))
	b = bytes.ReplaceAll(b, []byte(templateTokenEncoded), []byte(newTokenEncoded))
	b = bytes.ReplaceAll(b, []byte(templateCurrencyID), newCurrencyID.Bytes())

	{ // add new item
		r := currency.MustAddress(util.UUID().String())

		item := currency.NewTransfersItemSingleAmount(r, currency.MustNewAmount(currency.NewBig(10), t.cid))
		ib, err := jsonenc.Marshal(item)
		t.NoError(err)
		b = bytes.ReplaceAll(
			b,
			[]byte(`"items":[`),
			[]byte(`"items":[`+string(ib)+","),
		)
	}

	uhal, err := bl.BuildFact(b)
	t.NoError(err)

	uop, ok := uhal.Interface().(currency.Transfers)
	t.True(ok)
	err = uop.IsValid(nil)
	t.Contains(err.Error(), "malformed signature")

	ufact := uop.Fact().(currency.TransfersFact)

	t.True(ufact.Sender().Equal(newSender))
	t.True(ufact.Items()[1].Receiver().Equal(newReceiver))
	t.Equal(ufact.Token(), newToken)
	t.Equal(1, len(ufact.Items()[1].Amounts()))
	t.Equal(newBig, ufact.Items()[1].Amounts()[0].Big())
	t.Equal(newCurrencyID, ufact.Items()[1].Amounts()[0].Currency())

	sb, found := uhal.Extras()["signature_base"]
	t.True(found)

	_ = t.buildOperation(uop, sb.([]byte))
}

func (t *testBuilder) TestBuildFactCurrencyRegister() {
	bl := NewBuilder(t.JSONEnc, t.networkID)

	hal, err := bl.FactTemplate(currency.CurrencyRegisterHinter.Hint())
	t.NoError(err)

	b, err := t.JSONEnc.Marshal(hal)
	t.NoError(err)
	rhal := t.decodeHal(b)

	templateTokenEncoded := base64.StdEncoding.EncodeToString(templateToken)

	newSender := currency.NewAddress("new-mother")
	newReceiver := currency.NewAddress("new-father")
	newBig := currency.NewBig(99)
	newToken := util.UUID().Bytes()
	newTokenEncoded := base64.StdEncoding.EncodeToString(newToken)
	newCurrencyID := currency.CurrencyID("XXX")

	b = bytes.ReplaceAll(rhal.RawInterface(), []byte(templateSender.String()), []byte(newSender.String()))
	b = bytes.ReplaceAll(b, []byte(templateReceiver.String()), []byte(newReceiver.String()))
	b = bytes.ReplaceAll(b, []byte(templateBig.String()), []byte(newBig.String()))
	b = bytes.ReplaceAll(b, []byte(templateTokenEncoded), []byte(newTokenEncoded))
	b = bytes.ReplaceAll(b, []byte(templateCurrencyID), newCurrencyID.Bytes())

	uhal, err := bl.BuildFact(b)
	t.NoError(err)

	uop, ok := uhal.Interface().(currency.CurrencyRegister)
	t.True(ok)
	err = uop.IsValid(nil)
	t.Contains(err.Error(), "malformed signature")

	ufact := uop.Fact().(currency.CurrencyRegisterFact)

	t.Equal(ufact.Token(), newToken)
	t.True(ufact.Currency().GenesisAccount().Equal(newReceiver))
	t.Equal(ufact.Currency().Currency(), newCurrencyID)
	t.Equal(ufact.Currency().Big(), newBig)
	t.Equal(ufact.Currency().Policy().NewAccountMinBalance(), newBig)
	t.Equal(currency.NewNilFeeer(), ufact.Currency().Policy().Feeer())

	sb, found := uhal.Extras()["signature_base"]
	t.True(found)

	_ = t.buildOperation(uop, sb.([]byte))
}

func (t *testBuilder) TestBuildFactCurrencyPolicyUpdater() {
	bl := NewBuilder(t.JSONEnc, t.networkID)

	hal, err := bl.FactTemplate(currency.CurrencyPolicyUpdaterHinter.Hint())
	t.NoError(err)

	b, err := t.JSONEnc.Marshal(hal)
	t.NoError(err)
	rhal := t.decodeHal(b)

	templateTokenEncoded := base64.StdEncoding.EncodeToString(templateToken)

	newSender := currency.NewAddress("new-mother")
	newReceiver := currency.NewAddress("new-father")
	newBig := currency.NewBig(99)
	newToken := util.UUID().Bytes()
	newTokenEncoded := base64.StdEncoding.EncodeToString(newToken)
	newCurrencyID := currency.CurrencyID("XXX")

	b = bytes.ReplaceAll(rhal.RawInterface(), []byte(templateSender.String()), []byte(newSender.String()))
	b = bytes.ReplaceAll(b, []byte(templateReceiver.String()), []byte(newReceiver.String()))
	b = bytes.ReplaceAll(b, []byte(templateBig.String()), []byte(newBig.String()))
	b = bytes.ReplaceAll(b, []byte(templateTokenEncoded), []byte(newTokenEncoded))
	b = bytes.ReplaceAll(b, []byte(templateCurrencyID), newCurrencyID.Bytes())

	uhal, err := bl.BuildFact(b)
	t.NoError(err)

	uop, ok := uhal.Interface().(currency.CurrencyPolicyUpdater)
	t.True(ok)
	err = uop.IsValid(nil)
	t.Contains(err.Error(), "malformed signature")

	ufact := uop.Fact().(currency.CurrencyPolicyUpdaterFact)

	t.Equal(ufact.Token(), newToken)
	t.Equal(ufact.Currency(), newCurrencyID)
	t.Equal(ufact.Policy().NewAccountMinBalance(), newBig)
	t.Equal(currency.NewNilFeeer(), ufact.Policy().Feeer())

	sb, found := uhal.Extras()["signature_base"]
	t.True(found)

	_ = t.buildOperation(uop, sb.([]byte))
}

func (t *testBuilder) buildOperation(op operation.Operation, sb []byte) operation.Operation {
	priv := key.NewBasePrivatekey()
	sig, err := priv.Sign(sb)
	t.NoError(err)

	b, err := t.JSONEnc.Marshal(op)
	t.NoError(err)

	b = bytes.ReplaceAll(
		b,
		[]byte(templatePublickey.String()),
		[]byte(priv.Publickey().String()),
	)
	b = bytes.ReplaceAll(
		b,
		[]byte(base58.Encode(templateSignature)),
		[]byte(base58.Encode(sig)),
	)

	npriv := key.NewBasePrivatekey()
	nsig, err := npriv.Sign(sb)
	var nfs base.FactSign
	{ // add new FactSign
		t.NoError(err)
		nfs = base.RawBaseFactSign(npriv.Publickey(), nsig, time.Now())

		ib, err := jsonenc.Marshal(nfs)
		t.NoError(err)

		b = bytes.ReplaceAll(
			b,
			[]byte(`"fact_signs":[`),
			[]byte(`"fact_signs":[`+string(ib)+","),
		)
	}

	bl := NewBuilder(t.JSONEnc, t.networkID)

	fhal, err := bl.BuildOperation(b)
	t.NoError(err)

	fop, ok := fhal.Interface().(operation.Operation)
	t.True(ok)

	t.NoError(fop.IsValid(t.networkID))

	// new added factsign is not changed
	t.True(npriv.Publickey().Equal(fop.Signs()[0].Signer()))
	t.Equal(nsig, fop.Signs()[0].Signature())
	t.True(localtime.Equal(nfs.SignedAt(), fop.Signs()[0].SignedAt()))

	t.True(priv.Publickey().Equal(fop.Signs()[1].Signer()))
	t.Equal(sig, fop.Signs()[1].Signature())

	return fop
}

func (t *testBuilder) TestUpdateToken() {
	bl := NewBuilder(nil, nil)

	_, err := bl.checkToken(nil)
	t.Contains(err.Error(), "empty token")

	ntoken, err := bl.checkToken(templateToken)
	t.NoError(err)
	t.NotEmpty(ntoken)
	t.NotEqual(templateToken, ntoken)
}

func TestBuilder(t *testing.T) {
	suite.Run(t, new(testBuilder))
}
