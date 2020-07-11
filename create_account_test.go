package mc

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

type testCreateAcount struct {
	suite.Suite
}

func (t *testCreateAcount) TestNew() {
	spk := key.MustNewBTCPrivatekey()
	rpk := key.MustNewBTCPrivatekey()

	skey := NewKey(spk.Publickey(), 50)
	rkey := NewKey(rpk.Publickey(), 50)
	skeys, _ := NewKeys([]Key{skey, rkey}, 100)

	pks := []key.Privatekey{spk, rpk}
	sender, _ := NewAddressFromKeys([]Key{skey})

	token := util.UUID().Bytes()
	fact := NewCreateAccountFact(token, sender, skeys, NewAmount(10))

	var fs []operation.FactSign

	for _, pk := range pks {
		sig, err := operation.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
	}

	ca, err := NewCreateAccount(fact, fs)
	t.NoError(err)

	t.NoError(ca.IsValid(nil))

	t.Implements((*base.Fact)(nil), ca.Fact())
	t.Implements((*operation.Operation)(nil), ca)
}

func TestCreateAcount(t *testing.T) {
	suite.Run(t, new(testCreateAcount))
}

func testCreateAccountEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestOperationEncode)

	t.enc = enc
	t.newOperation = func() operation.Operation {
		spk := key.MustNewBTCPrivatekey()
		rpk := key.MustNewBTCPrivatekey()

		skey := NewKey(spk.Publickey(), 50)
		skeys, _ := NewKeys([]Key{skey, NewKey(rpk.Publickey(), 50)}, 100)

		pks := []key.Privatekey{spk, rpk}
		sender, _ := NewAddressFromKeys([]Key{skey})

		fact := NewCreateAccountFact(util.UUID().Bytes(), sender, skeys, NewAmount(10))

		var fs []operation.FactSign

		for _, pk := range pks {
			sig, err := operation.NewFactSignature(pk, fact, nil)
			t.NoError(err)

			fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
		}

		ca, err := NewCreateAccount(fact, fs)
		t.NoError(err)

		return ca
	}

	t.compare = func(a, b operation.Operation) {
		fact := a.Fact().(CreateAccountFact)
		ufact := b.Fact().(CreateAccountFact)

		t.True(fact.sender.Equal(ufact.sender))
		t.Equal(fact.amount, ufact.amount)

		t.True(fact.keys.Hash().Equal(ufact.keys.Hash()))
		t.Equal(fact.keys.Keys(), ufact.keys.Keys())
		t.Equal(fact.keys.Threshold(), ufact.keys.Threshold())
	}

	return t
}

func TestCreateAccountEncodeJSON(t *testing.T) {
	suite.Run(t, testCreateAccountEncode(jsonenc.NewEncoder()))
}

func TestCreateAccountEncodeBSON(t *testing.T) {
	suite.Run(t, testCreateAccountEncode(bsonenc.NewEncoder()))
}
