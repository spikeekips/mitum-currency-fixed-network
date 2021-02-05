package currency

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
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

	am := NewAmount(NewBig(11), CurrencyID("SHOWME"))

	item := NewCreateAccountsItemSingleAmount(skeys, am)
	fact := NewCreateAccountsFact(token, sender, []CreateAccountsItem{item})

	var fs []operation.FactSign

	for _, pk := range pks {
		sig, err := operation.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
	}

	op, err := NewCreateAccounts(fact, fs, "")
	t.NoError(err)

	t.NoError(op.IsValid(nil))

	t.Implements((*base.Fact)(nil), op.Fact())
	t.Implements((*operation.Operation)(nil), op)

	ufact := op.Fact().(CreateAccountsFact)
	t.Equal(1, len(ufact.Items()[0].Amounts()))
	t.Equal(am, ufact.Items()[0].Amounts()[0])
}

func (t *testCreateAccounts) TestDuplicatedKeys() {
	var items []CreateAccountsItem
	{
		pk := key.MustNewBTCPrivatekey()
		key, err := NewKey(pk.Publickey(), 100)
		t.NoError(err)
		keys, err := NewKeys([]Key{key}, 100)
		t.NoError(err)

		items = append(items, NewCreateAccountsItemSingleAmount(keys, NewAmount(NewBig(11), CurrencyID("FINDME"))))
		items = append(items, NewCreateAccountsItemSingleAmount(keys, NewAmount(NewBig(33), CurrencyID("SHOWME"))))
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

	op, err := NewCreateAccounts(fact, fs, "")
	t.NoError(err)

	err = op.IsValid(nil)
	t.Contains(err.Error(), "duplicated acocunt Keys found")
}

func (t *testCreateAccounts) TestSameWithSender() {
	pk := key.MustNewBTCPrivatekey()
	key, err := NewKey(pk.Publickey(), 100)
	t.NoError(err)
	keys, err := NewKeys([]Key{key}, 100)
	t.NoError(err)

	am := NewAmount(NewBig(11), CurrencyID("SHOWME"))
	items := []CreateAccountsItem{NewCreateAccountsItemSingleAmount(keys, am)}

	token := util.UUID().Bytes()
	sender, _ := NewAddressFromKeys(keys)
	fact := NewCreateAccountsFact(token, sender, items)

	sig, err := operation.NewFactSignature(pk, fact, nil)
	t.NoError(err)
	fs := []operation.FactSign{operation.NewBaseFactSign(pk.Publickey(), sig)}

	op, err := NewCreateAccounts(fact, fs, "")
	t.NoError(err)

	err = op.IsValid(nil)
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

	am := NewAmount(NewBig(11), CurrencyID("SHOWME"))
	item := NewCreateAccountsItemSingleAmount(skeys, am)
	fact := NewCreateAccountsFact(token, sender, []CreateAccountsItem{item})

	var fs []operation.FactSign

	for _, pk := range pks {
		sig, err := operation.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
	}

	memo := strings.Repeat("a", MaxMemoSize) + "a"
	op, err := NewCreateAccounts(fact, fs, memo)
	t.NoError(err)

	err = op.IsValid(nil)
	t.Contains(err.Error(), "memo over max size")
}

func TestCreateAccounts(t *testing.T) {
	suite.Run(t, new(testCreateAccounts))
}
