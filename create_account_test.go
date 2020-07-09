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
	"github.com/spikeekips/mitum/util/localtime"
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

type testCreateAccountEncode struct {
	suite.Suite

	enc encoder.Encoder
}

func (t *testCreateAccountEncode) SetupSuite() {
	encs := encoder.NewEncoders()
	encs.AddEncoder(t.enc)

	encs.AddHinter(key.BTCPublickey{})
	encs.AddHinter(Address(""))
	encs.AddHinter(operation.BaseFactSign{})

	encs.AddHinter(Key{})
	encs.AddHinter(Keys{})
	encs.AddHinter(CreateAccountFact{})
	encs.AddHinter(CreateAccount{})
}

func (t *testCreateAccountEncode) TestEncode() {
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

	b, err := t.enc.Marshal(ca)
	t.NoError(err)

	hinter, err := t.enc.DecodeByHint(b)
	t.NoError(err)

	uca, ok := hinter.(CreateAccount)
	t.True(ok)

	ufact := uca.Fact().(CreateAccountFact)
	t.True(fact.h.Equal(ufact.h))
	t.Equal(fact.token, ufact.token)
	t.True(fact.sender.Equal(ufact.sender))
	t.Equal(fact.amount, ufact.amount)

	t.True(ca.Hash().Equal(uca.Hash()))

	t.True(fact.keys.Hash().Equal(ufact.keys.Hash()))
	t.Equal(fact.keys.Keys(), ufact.keys.Keys())
	t.Equal(fact.keys.Threshold(), ufact.keys.Threshold())

	for i := range ca.Signs() {
		a := ca.Signs()[i]
		b := uca.Signs()[i]

		t.True(a.Signer().Equal(b.Signer()))
		t.Equal(a.Signature(), b.Signature())
		t.Equal(localtime.RFC3339(a.SignedAt()), localtime.RFC3339(b.SignedAt()))
	}
}

func TestCreateAccountEncodeJSON(t *testing.T) {
	b := new(testCreateAccountEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func testCreateAccountEncodeBSON(t *testing.T) {
	b := new(testCreateAccountEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
