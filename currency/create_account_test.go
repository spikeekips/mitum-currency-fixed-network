package currency

import (
	"strings"
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

type testCreateAccounts struct {
	baseTest
}

func (t *testCreateAccounts) TestNew() {
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

	item := NewCreateAccountItem(skeys, NewAmount(10))
	fact := NewCreateAccountsFact(token, sender, []CreateAccountItem{item})

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
}

func (t *testCreateAccounts) TestZeroAmount() {
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

	item := NewCreateAccountItem(skeys, NewAmount(0))
	err = item.IsValid(nil)
	t.Contains(err.Error(), "amount should be over zero")

	fact := NewCreateAccountsFact(token, sender, []CreateAccountItem{item})

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

func (t *testCreateAccounts) TestDuplicatedKeys() {
	var items []CreateAccountItem
	{
		pk := key.MustNewBTCPrivatekey()
		key, err := NewKey(pk.Publickey(), 100)
		t.NoError(err)
		keys, err := NewKeys([]Key{key}, 100)
		t.NoError(err)

		items = append(items, NewCreateAccountItem(keys, NewAmount(10)))
		items = append(items, NewCreateAccountItem(keys, NewAmount(30)))
	}

	token := util.UUID().Bytes()
	pk := key.MustNewBTCPrivatekey()
	key, err := NewKey(pk.Publickey(), 100)
	t.NoError(err)

	keys, _ := NewKeys([]Key{key}, 100)
	sender, _ := NewAddressFromKeys(keys)
	fact := NewCreateAccountsFact(token, sender, items)

	sig, err := operation.NewFactSignature(pk, fact, nil)
	t.NoError(err)
	fs := []operation.FactSign{operation.NewBaseFactSign(pk.Publickey(), sig)}

	ca, err := NewCreateAccounts(fact, fs, "")
	t.NoError(err)

	err = ca.IsValid(nil)
	t.Contains(err.Error(), "duplicated acocunt Keys found")
}

func (t *testCreateAccounts) TestSameWithSender() {
	pk := key.MustNewBTCPrivatekey()
	key, err := NewKey(pk.Publickey(), 100)
	t.NoError(err)
	keys, err := NewKeys([]Key{key}, 100)
	t.NoError(err)

	items := []CreateAccountItem{NewCreateAccountItem(keys, NewAmount(10))}

	token := util.UUID().Bytes()
	sender, _ := NewAddressFromKeys(keys)
	fact := NewCreateAccountsFact(token, sender, items)

	sig, err := operation.NewFactSignature(pk, fact, nil)
	t.NoError(err)
	fs := []operation.FactSign{operation.NewBaseFactSign(pk.Publickey(), sig)}

	ca, err := NewCreateAccounts(fact, fs, "")
	t.NoError(err)

	err = ca.IsValid(nil)
	t.Contains(err.Error(), "target address is same with sender")
}

func (t *testCreateAccounts) TestOverSizeMemo() {
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

	token := util.UUID().Bytes()

	item := NewCreateAccountItem(skeys, NewAmount(10))
	fact := NewCreateAccountsFact(token, sender, []CreateAccountItem{item})

	var fs []operation.FactSign

	for _, pk := range pks {
		sig, err := operation.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
	}

	memo := strings.Repeat("a", MaxMemoSize) + "a"
	ca, err := NewCreateAccounts(fact, fs, memo)
	t.NoError(err)

	err = ca.IsValid(nil)
	t.Contains(err.Error(), "memo over max size")
}

func TestCreateAccounts(t *testing.T) {
	suite.Run(t, new(testCreateAccounts))
}

func testCreateAccountsEncode(enc encoder.Encoder) suite.TestingSuite {
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

		item := NewCreateAccountItem(skeys, NewAmount(10))
		fact := NewCreateAccountsFact(util.UUID().Bytes(), sender, []CreateAccountItem{item})

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
		t.Equal(fact.Amount(), ufact.Amount())
		t.Equal(len(fact.Items()), len(ufact.Items()))

		for i := range fact.Items() {
			a := fact.Items()[i]
			b := ufact.Items()[i]

			t.True(a.keys.Hash().Equal(b.keys.Hash()))
			for i := range a.keys.Keys() {
				t.Equal(a.keys.Keys()[i].Bytes(), b.keys.Keys()[i].Bytes())
			}

			t.Equal(a.keys.Threshold(), b.keys.Threshold())
		}
	}

	return t
}

func TestCreateAccountsEncodeJSON(t *testing.T) {
	suite.Run(t, testCreateAccountsEncode(jsonenc.NewEncoder()))
}

func TestCreateAccountsEncodeBSON(t *testing.T) {
	suite.Run(t, testCreateAccountsEncode(bsonenc.NewEncoder()))
}
