// +build mongodb

package digest

import (
	"bytes"
	"encoding/base64"
	"testing"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/stretchr/testify/suite"
)

type testBuilder struct {
	baseTest
}

func (t *testBuilder) decodeHal(b []byte) Hal {
	hinter, err := t.JSONEnc.DecodeByHint(b)
	if err != nil {
		panic(err)
	}

	hal, ok := hinter.(Hal)
	if !ok {
		panic("not Hal")
	}

	hinter, err = t.JSONEnc.DecodeByHint(hal.RawInterface())
	if err != nil {
		panic(err)
	}

	return hal.SetInterface(hinter)
}

func (t *testBuilder) TestUnknownFactTemplate() {
	bl := NewBuilder(t.JSONEnc, t.networkID)

	_, err := bl.FactTemplate(currency.CreateAccounts{}.Hint())
	t.NoError(err)

	_, err = bl.FactTemplate(currency.KeyUpdater{}.Hint())
	t.NoError(err)

	_, err = bl.FactTemplate(currency.Transfers{}.Hint())
	t.NoError(err)

	_, err = bl.FactTemplate(currency.GenesisAccount{}.Hint())
	t.Contains(err.Error(), "unknown operation")
}

func (t *testBuilder) TestFactTemplateCreateAccounts() {
	bl := NewBuilder(t.JSONEnc, t.networkID)

	hal, err := bl.FactTemplate(currency.CreateAccounts{}.Hint())
	t.NoError(err)
	t.NotEmpty(hal.Extras())

	b, err := t.JSONEnc.Marshal(hal)
	t.NoError(err)
	uhal := t.decodeHal(b)

	t.IsType(currency.CreateAccountsFact{}, uhal.Interface())
}

func (t *testBuilder) TestFactTemplateKeyUpdater() {
	bl := NewBuilder(t.JSONEnc, t.networkID)

	hal, err := bl.FactTemplate(currency.KeyUpdater{}.Hint())
	t.NoError(err)
	t.NotEmpty(hal.Extras())

	b, err := t.JSONEnc.Marshal(hal)
	t.NoError(err)
	uhal := t.decodeHal(b)

	t.IsType(currency.KeyUpdaterFact{}, uhal.Interface())
}

func (t *testBuilder) TestFactTemplateTransfers() {
	bl := NewBuilder(t.JSONEnc, t.networkID)

	hal, err := bl.FactTemplate(currency.Transfers{}.Hint())
	t.NoError(err)
	t.NotEmpty(hal.Extras())

	b, err := t.JSONEnc.Marshal(hal)
	t.NoError(err)
	uhal := t.decodeHal(b)

	t.IsType(currency.TransfersFact{}, uhal.Interface())
}

func (t *testBuilder) TestBuildFactCreateAccounts() {
	bl := NewBuilder(t.JSONEnc, t.networkID)

	hal, err := bl.FactTemplate(currency.CreateAccounts{}.Hint())

	b, err := t.JSONEnc.Marshal(hal)
	t.NoError(err)
	rhal := t.decodeHal(b)

	templateTokenEncoded := base64.StdEncoding.EncodeToString(templateToken)

	newPub := key.MustNewBTCPrivatekey().Publickey()
	newSender := currency.Address("new-mother")
	newAmount := currency.NewAmount(99)
	newToken := util.UUID().Bytes()
	newTokenEncoded := base64.StdEncoding.EncodeToString(newToken)

	b = bytes.ReplaceAll(
		rhal.RawInterface(),
		[]byte(templateSender.HintedString()),
		[]byte(newSender.HintedString()),
	)
	b = bytes.ReplaceAll(
		b,
		[]byte(hint.HintedString(templatePublickey.Hint(), templatePublickey.String())),
		[]byte(hint.HintedString(newPub.Hint(), newPub.String())),
	)
	b = bytes.ReplaceAll(
		b,
		[]byte(templateAmount.String()),
		[]byte(newAmount.String()),
	)
	b = bytes.ReplaceAll(
		b,
		[]byte(templateTokenEncoded),
		[]byte(newTokenEncoded),
	)

	uhal, err := bl.BuildFact(b)
	t.NoError(err)

	uop, ok := uhal.Interface().(currency.CreateAccounts)
	t.True(ok)
	err = uop.IsValid(nil)
	t.Contains(err.Error(), "malformed signature")

	ufact := uop.Fact().(currency.CreateAccountsFact)

	t.True(ufact.Sender().Equal(newSender))
	t.Equal(ufact.Token(), newToken)
	t.Equal(newAmount, ufact.Items()[0].Amount())

	_, same := ufact.Items()[0].Keys().Key(newPub)
	t.True(same)

	sb, found := uhal.Extras()["signature_base"]
	t.True(found)

	_ = t.buildOperation(uop, sb.([]byte))
}

func (t *testBuilder) TestBuildFactKeyUpdater() {
	bl := NewBuilder(t.JSONEnc, t.networkID)

	hal, err := bl.FactTemplate(currency.KeyUpdater{}.Hint())

	b, err := t.JSONEnc.Marshal(hal)
	t.NoError(err)
	rhal := t.decodeHal(b)

	templateTokenEncoded := base64.StdEncoding.EncodeToString(templateToken)

	newPub := key.MustNewBTCPrivatekey().Publickey()
	newSender := currency.Address("new-mother")
	newToken := util.UUID().Bytes()
	newTokenEncoded := base64.StdEncoding.EncodeToString(newToken)

	b = bytes.ReplaceAll(
		rhal.RawInterface(),
		[]byte(templateSender.HintedString()),
		[]byte(newSender.HintedString()),
	)
	b = bytes.ReplaceAll(
		b,
		[]byte(hint.HintedString(templatePublickey.Hint(), templatePublickey.String())),
		[]byte(hint.HintedString(newPub.Hint(), newPub.String())),
	)
	b = bytes.ReplaceAll(
		b,
		[]byte(templateTokenEncoded),
		[]byte(newTokenEncoded),
	)

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

	sb, found := uhal.Extras()["signature_base"]
	t.True(found)

	_ = t.buildOperation(uop, sb.([]byte))
}

func (t *testBuilder) TestBuildFactTransfers() {
	bl := NewBuilder(t.JSONEnc, t.networkID)

	hal, err := bl.FactTemplate(currency.Transfers{}.Hint())

	b, err := t.JSONEnc.Marshal(hal)
	t.NoError(err)
	rhal := t.decodeHal(b)

	templateTokenEncoded := base64.StdEncoding.EncodeToString(templateToken)

	newSender := currency.Address("new-mother")
	newReceiver := currency.Address("new-father")
	newAmount := currency.NewAmount(99)
	newToken := util.UUID().Bytes()
	newTokenEncoded := base64.StdEncoding.EncodeToString(newToken)

	b = bytes.ReplaceAll(
		rhal.RawInterface(),
		[]byte(templateSender.HintedString()),
		[]byte(newSender.HintedString()),
	)
	b = bytes.ReplaceAll(
		b,
		[]byte(templateReceiver.HintedString()),
		[]byte(newReceiver.HintedString()),
	)
	b = bytes.ReplaceAll(
		b,
		[]byte(templateAmount.String()),
		[]byte(newAmount.String()),
	)
	b = bytes.ReplaceAll(
		b,
		[]byte(templateTokenEncoded),
		[]byte(newTokenEncoded),
	)

	{ // add new item
		r := currency.MustAddress(util.UUID().String())

		item := currency.NewTransferItem(r, currency.NewAmount(10))
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
	t.Equal(newAmount, ufact.Items()[1].Amount())

	sb, found := uhal.Extras()["signature_base"]
	t.True(found)

	_ = t.buildOperation(uop, sb.([]byte))
}

func (t *testBuilder) buildOperation(op operation.Operation, sb []byte) operation.Operation {
	priv := key.MustNewBTCPrivatekey()
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

	npriv := key.MustNewBTCPrivatekey()
	nsig, err := npriv.Sign(sb)
	var nfs operation.FactSign
	{ // add new FactSign
		t.NoError(err)
		nfs = operation.RawBaseFactSign(npriv.Publickey(), nsig, time.Now())

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
	t.Equal(localtime.RFC3339(nfs.SignedAt()), localtime.RFC3339(fop.Signs()[0].SignedAt()))

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
