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

type testTransfer struct {
	suite.Suite
}

func (t *testTransfer) TestNew() {
	s := MustAddress(util.UUID().String())
	r := MustAddress(util.UUID().String())

	token := util.UUID().Bytes()
	fact := NewTransferFact(token, s, r, NewAmount(10))

	var fs []operation.FactSign

	for _, pk := range []key.Privatekey{
		key.MustNewBTCPrivatekey(),
		key.MustNewBTCPrivatekey(),
		key.MustNewBTCPrivatekey(),
	} {
		sig, err := operation.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
	}

	tf, err := NewTransfer(fact, fs)
	t.NoError(err)

	t.NoError(tf.IsValid(nil))

	t.Implements((*base.Fact)(nil), tf.Fact())
	t.Implements((*operation.Operation)(nil), tf)
}

func TestTransfer(t *testing.T) {
	suite.Run(t, new(testTransfer))
}

type testTransferEncode struct {
	suite.Suite

	enc encoder.Encoder
}

func (t *testTransferEncode) SetupSuite() {
	encs := encoder.NewEncoders()
	encs.AddEncoder(t.enc)

	encs.AddHinter(TransferFact{})
	encs.AddHinter(Transfer{})
	encs.AddHinter(key.BTCPublickey{})
	encs.AddHinter(Address(""))
	encs.AddHinter(operation.BaseFactSign{})
}

func (t *testTransferEncode) TestEncode() {
	s := MustAddress(util.UUID().String())
	r := MustAddress(util.UUID().String())

	token := util.UUID().Bytes()
	fact := NewTransferFact(token, s, r, NewAmount(10))

	var fs []operation.FactSign

	for _, pk := range []key.Privatekey{
		key.MustNewBTCPrivatekey(),
		key.MustNewBTCPrivatekey(),
		key.MustNewBTCPrivatekey(),
	} {
		sig, err := operation.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
	}

	tf, err := NewTransfer(fact, fs)
	t.NoError(err)

	b, err := t.enc.Marshal(tf)
	t.NoError(err)

	hinter, err := t.enc.DecodeByHint(b)
	t.NoError(err)

	utf, ok := hinter.(Transfer)
	t.True(ok)

	ufact := utf.Fact().(TransferFact)
	t.True(fact.h.Equal(ufact.h))
	t.Equal(fact.token, ufact.token)
	t.True(fact.sender.Equal(ufact.sender))
	t.True(fact.receiver.Equal(ufact.receiver))
	t.Equal(fact.amount, ufact.amount)

	t.True(tf.Hash().Equal(utf.Hash()))

	for i := range tf.Signs() {
		a := tf.Signs()[i]
		b := utf.Signs()[i]

		t.True(a.Signer().Equal(b.Signer()))
		t.Equal(a.Signature(), b.Signature())
		t.Equal(localtime.RFC3339(a.SignedAt()), localtime.RFC3339(b.SignedAt()))
	}
}

func TestTransferEncodeJSON(t *testing.T) {
	b := new(testTransferEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestTransferEncodeBSON(t *testing.T) {
	b := new(testTransferEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
