package currency

import (
	"fmt"
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

type testCreateAccountsMultiAmounts struct {
	baseTest
}

func (t *testCreateAccountsMultiAmounts) TestNew() {
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

	ams := []Amount{
		NewAmount(NewBig(11), CurrencyID("SHOWME")),
		NewAmount(NewBig(22), CurrencyID("FINDME")),
	}

	item := NewCreateAccountsItemMultiAmounts(skeys, ams)
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
	t.Equal(2, len(ufact.Items()[0].Amounts()))
	t.Equal(ams, ufact.Items()[0].Amounts())
}

func (t *testCreateAccountsMultiAmounts) TestZeroBig() {
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

	ams := []Amount{
		NewAmount(NewBig(0), CurrencyID("SHOWME")),
		NewAmount(NewBig(22), CurrencyID("FINDME")),
	}

	item := NewCreateAccountsItemMultiAmounts(skeys, ams)
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

func (t *testCreateAccountsMultiAmounts) TestEmptyAmounts() {
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

	item := NewCreateAccountsItemMultiAmounts(skeys, nil)
	err = item.IsValid(nil)
	t.Contains(err.Error(), "empty amounts")

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
	t.Contains(err.Error(), "empty amounts")
}

func (t *testCreateAccountsMultiAmounts) TestOverMaxAmounts() {
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

	var ams []Amount
	for i := 0; i < maxCurenciesCreateAccountsItemMultiAmounts+1; i++ {
		ams = append(ams, NewAmount(NewBig(11), CurrencyID(fmt.Sprintf("FINDME_%d", i))))
	}

	item := NewCreateAccountsItemMultiAmounts(skeys, ams)
	err = item.IsValid(nil)
	t.Contains(err.Error(), "amounts over allowed")

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
	t.Contains(err.Error(), "amounts over allowed")
}

func (t *testCreateAccountsMultiAmounts) TestDuplicatedCurrency() {
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

	ams := []Amount{
		NewAmount(NewBig(11), CurrencyID("FINDME")),
		NewAmount(NewBig(22), CurrencyID("FINDME")),
	}

	item := NewCreateAccountsItemMultiAmounts(skeys, ams)
	err = item.IsValid(nil)
	t.Contains(err.Error(), "duplicated currency found")

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
	t.Contains(err.Error(), "duplicated currency found")
}

func TestCreateAccountsMultiAmounts(t *testing.T) {
	suite.Run(t, new(testCreateAccountsMultiAmounts))
}

func testCreateAccountsMultiAmountsEncode(enc encoder.Encoder) suite.TestingSuite {
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

		ams := []Amount{
			NewAmount(NewBig(11), CurrencyID("SHOWME")),
			NewAmount(NewBig(22), CurrencyID("FINDME")),
		}

		item := NewCreateAccountsItemMultiAmounts(skeys, ams)
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

func TestCreateAccountsMultiAmountsEncodeJSON(t *testing.T) {
	suite.Run(t, testCreateAccountsMultiAmountsEncode(jsonenc.NewEncoder()))
}

func TestCreateAccountsMultiAmountsEncodeBSON(t *testing.T) {
	suite.Run(t, testCreateAccountsMultiAmountsEncode(bsonenc.NewEncoder()))
}
