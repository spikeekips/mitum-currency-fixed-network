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

	tf, err := NewTransfer(fact, fs, "")
	t.NoError(err)

	t.NoError(tf.IsValid(nil))

	t.Implements((*base.Fact)(nil), tf.Fact())
	t.Implements((*operation.Operation)(nil), tf)
}

func (t *testTransfer) TestOverSizeMemo() {
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

	memo := strings.Repeat("a", MaxMemoSize) + "a"
	tf, err := NewTransfer(fact, fs, memo)
	t.NoError(err)

	err = tf.IsValid(nil)
	t.Contains(err.Error(), "memo over max size")
}

func TestTransfer(t *testing.T) {
	suite.Run(t, new(testTransfer))
}

func testTransferEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestOperationEncode)

	t.enc = enc
	t.newOperation = func() operation.Operation {
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

		tf, err := NewTransfer(fact, fs, util.UUID().String())
		t.NoError(err)

		return tf
	}

	t.compare = func(a, b operation.Operation) {
		ta := a.(Transfer)
		tb := b.(Transfer)

		t.Equal(ta.Memo, tb.Memo)

		fact := a.Fact().(TransferFact)
		ufact := b.Fact().(TransferFact)

		t.True(fact.sender.Equal(ufact.sender))
		t.True(fact.receiver.Equal(ufact.receiver))
		t.Equal(fact.amount, ufact.amount)
	}

	return t
}

func TestTransferEncodeJSON(t *testing.T) {
	suite.Run(t, testTransferEncode(jsonenc.NewEncoder()))
}

func TestTransferEncodeBSON(t *testing.T) {
	suite.Run(t, testTransferEncode(bsonenc.NewEncoder()))
}
