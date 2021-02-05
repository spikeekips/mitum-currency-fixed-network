package currency

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type testCreateAccountsSingleAmount struct {
	baseTest
}

func (t *testCreateAccountsSingleAmount) TestNew() {
	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey, err := NewKey(spk.Publickey(), 50)
	t.NoError(err)
	rkey, err := NewKey(rpk.Publickey(), 50)
	t.NoError(err)
	skeys, _ := NewKeys([]Key{skey, rkey}, 100)

	pks := []key.Privatekey{spk, rpk}

	keys, _ := NewKeys([]Key{skey}, 50)
	sender, _ := NewAddressFromKeys(keys)

	token := util.UUID().Bytes()

	am := NewAmount(NewBig(11), CurrencyID("SHOWME"))

	item := NewCreateAccountsItemSingleAmount(skeys, am)
	fact := NewCreateAccountsFact(token, sender, []CreateAccountsItem{item})

	var fs []operation.FactSign

	for _, pk := range pks {
		sig, err := operation.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
	}

	ca, err := NewCreateAccounts(fact, fs, "")
	t.NoError(err)

	t.NoError(ca.IsValid(nil))

	t.Implements((*base.Fact)(nil), ca.Fact())
	t.Implements((*operation.Operation)(nil), ca)

	ufact := ca.Fact().(CreateAccountsFact)
	t.Equal(1, len(ufact.Items()[0].Amounts()))
	t.Equal(am, ufact.Items()[0].Amounts()[0])
}

func (t *testCreateAccountsSingleAmount) TestZeroBig() {
	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey, err := NewKey(spk.Publickey(), 50)
	t.NoError(err)
	rkey, err := NewKey(rpk.Publickey(), 50)
	t.NoError(err)
	skeys, _ := NewKeys([]Key{skey, rkey}, 100)

	pks := []key.Privatekey{spk, rpk}

	keys, _ := NewKeys([]Key{skey}, 50)
	sender, _ := NewAddressFromKeys(keys)

	token := util.UUID().Bytes()

	am := NewAmount(NewBig(0), CurrencyID("SHOWME"))

	item := NewCreateAccountsItemSingleAmount(skeys, am)
	err = item.IsValid(nil)
	t.Contains(err.Error(), "amount should be over zero")

	fact := NewCreateAccountsFact(token, sender, []CreateAccountsItem{item})

	var fs []operation.FactSign

	for _, pk := range pks {
		sig, err := operation.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
	}

	ca, err := NewCreateAccounts(fact, fs, "")
	t.NoError(err)

	err = ca.IsValid(nil)
	t.Contains(err.Error(), "amount should be over zero")
}

func (t *testCreateAccountsSingleAmount) TestEmptyAmounts() {
	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey, err := NewKey(spk.Publickey(), 50)
	t.NoError(err)
	rkey, err := NewKey(rpk.Publickey(), 50)
	t.NoError(err)
	skeys, _ := NewKeys([]Key{skey, rkey}, 100)

	pks := []key.Privatekey{spk, rpk}

	keys, _ := NewKeys([]Key{skey}, 50)
	sender, _ := NewAddressFromKeys(keys)

	token := util.UUID().Bytes()

	am := NewAmount(NewBig(0), CurrencyID("SHOWME"))

	item := NewCreateAccountsItemSingleAmount(skeys, am)
	item.amounts = nil
	err = item.IsValid(nil)
	t.Contains(err.Error(), "empty amount")

	fact := NewCreateAccountsFact(token, sender, []CreateAccountsItem{item})

	var fs []operation.FactSign

	for _, pk := range pks {
		sig, err := operation.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
	}

	ca, err := NewCreateAccounts(fact, fs, "")
	t.NoError(err)

	err = ca.IsValid(nil)
	t.Contains(err.Error(), "empty amount")
}

func (t *testCreateAccountsSingleAmount) TestTooManyAmounts() {
	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey, err := NewKey(spk.Publickey(), 50)
	t.NoError(err)
	rkey, err := NewKey(rpk.Publickey(), 50)
	t.NoError(err)
	skeys, _ := NewKeys([]Key{skey, rkey}, 100)

	pks := []key.Privatekey{spk, rpk}

	keys, _ := NewKeys([]Key{skey}, 50)
	sender, _ := NewAddressFromKeys(keys)

	token := util.UUID().Bytes()

	am := NewAmount(NewBig(0), CurrencyID("SHOWME"))

	item := NewCreateAccountsItemSingleAmount(skeys, am)
	item.amounts = []Amount{
		NewAmount(NewBig(11), CurrencyID("FINDME0")),
		NewAmount(NewBig(22), CurrencyID("FINDME1")),
	}

	err = item.IsValid(nil)
	t.Contains(err.Error(), "only one amount allowed")

	fact := NewCreateAccountsFact(token, sender, []CreateAccountsItem{item})

	var fs []operation.FactSign

	for _, pk := range pks {
		sig, err := operation.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
	}

	ca, err := NewCreateAccounts(fact, fs, "")
	t.NoError(err)

	err = ca.IsValid(nil)
	t.Contains(err.Error(), "only one amount allowed")
}

func TestCreateAccountsSingleAmount(t *testing.T) {
	suite.Run(t, new(testCreateAccountsSingleAmount))
}

func testCreateAccountsSingleAmountEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestOperationEncode)

	t.enc = enc
	t.newObject = func() interface{} {
		spk := key.MustNewBTCPrivatekey()
		rpk := key.MustNewBTCPrivatekey()

		skey, err := NewKey(spk.Publickey(), 50)
		t.NoError(err)
		rkey, err := NewKey(rpk.Publickey(), 50)
		t.NoError(err)
		skeys, err := NewKeys([]Key{skey, rkey}, 100)
		t.NoError(err)

		pks := []key.Privatekey{spk, rpk}
		sender, _ := NewAddressFromKeys(skeys)

		am := NewAmount(NewBig(11), CurrencyID("SHOWME"))

		item := NewCreateAccountsItemSingleAmount(skeys, am)
		fact := NewCreateAccountsFact(util.UUID().Bytes(), sender, []CreateAccountsItem{item})

		var fs []operation.FactSign

		for _, pk := range pks {
			sig, err := operation.NewFactSignature(pk, fact, nil)
			t.NoError(err)

			fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
		}

		ca, err := NewCreateAccounts(fact, fs, util.UUID().String())
		t.NoError(err)

		return ca
	}

	t.compare = func(a, b interface{}) {
		ca := a.(CreateAccounts)
		cb := b.(CreateAccounts)

		t.Equal(ca.Memo, cb.Memo)

		fact := ca.Fact().(CreateAccountsFact)
		ufact := cb.Fact().(CreateAccountsFact)

		t.True(fact.sender.Equal(ufact.sender))
		t.Equal(len(fact.Items()), len(ufact.Items()))

		for i := range fact.Items() {
			a := fact.Items()[i]
			b := ufact.Items()[i]

			t.True(a.Keys().Hash().Equal(b.Keys().Hash()))
			for i := range a.Keys().Keys() {
				t.Equal(a.Keys().Keys()[i].Bytes(), b.Keys().Keys()[i].Bytes())
			}

			t.Equal(a.Keys().Threshold(), b.Keys().Threshold())

			for j := range a.Amounts() {
				aam := a.Amounts()[j]
				bam := b.Amounts()[j]

				t.True(aam.Equal(bam))
			}
		}
	}

	return t
}

func TestCreateAccountsSingleAmountEncodeJSON(t *testing.T) {
	suite.Run(t, testCreateAccountsSingleAmountEncode(jsonenc.NewEncoder()))
}

func TestCreateAccountsSingleAmountEncodeBSON(t *testing.T) {
	suite.Run(t, testCreateAccountsSingleAmountEncode(bsonenc.NewEncoder()))
}
