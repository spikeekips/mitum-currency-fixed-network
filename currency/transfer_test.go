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

type testTransfers struct {
	suite.Suite
}

func (t *testTransfers) TestNew() {
	s := MustAddress(util.UUID().String())
	r := MustAddress(util.UUID().String())

	token := util.UUID().Bytes()
	items := []TransferItem{NewTransferItem(r, NewAmount(10))}
	fact := NewTransfersFact(token, s, items)

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

	tf, err := NewTransfers(fact, fs, "")
	t.NoError(err)

	t.NoError(tf.IsValid(nil))

	t.Implements((*base.Fact)(nil), tf.Fact())
	t.Implements((*operation.Operation)(nil), tf)
}

func (t *testTransfers) TestZeroAmount() {
	s := MustAddress(util.UUID().String())
	r := MustAddress(util.UUID().String())

	token := util.UUID().Bytes()
	items := []TransferItem{NewTransferItem(r, NewAmount(0))}

	err := items[0].IsValid(nil)
	t.Contains(err.Error(), "amount should be over zero")

	fact := NewTransfersFact(token, s, items)

	pk := key.MustNewBTCPrivatekey()
	sig, err := operation.NewFactSignature(pk, fact, nil)
	t.NoError(err)

	fs := []operation.FactSign{operation.NewBaseFactSign(pk.Publickey(), sig)}

	tf, err := NewTransfers(fact, fs, "")
	t.NoError(err)

	err = tf.IsValid(nil)
	t.Contains(err.Error(), "amount should be over zero")
}

func (t *testTransfers) TestDuplicatedReceivers() {
	s := MustAddress(util.UUID().String())
	r := MustAddress(util.UUID().String())

	token := util.UUID().Bytes()
	items := []TransferItem{
		NewTransferItem(r, NewAmount(1)),
		NewTransferItem(r, NewAmount(1)),
	}
	fact := NewTransfersFact(token, s, items)

	pk := key.MustNewBTCPrivatekey()
	sig, err := operation.NewFactSignature(pk, fact, nil)
	t.NoError(err)

	fs := []operation.FactSign{operation.NewBaseFactSign(pk.Publickey(), sig)}

	tf, err := NewTransfers(fact, fs, "")
	t.NoError(err)

	err = tf.IsValid(nil)
	t.Contains(err.Error(), "duplicated receiver found")
}

func (t *testTransfers) TestSameWithSender() {
	s := MustAddress(util.UUID().String())
	r := MustAddress(util.UUID().String())

	token := util.UUID().Bytes()
	items := []TransferItem{
		NewTransferItem(r, NewAmount(1)),
		NewTransferItem(s, NewAmount(1)),
	}
	fact := NewTransfersFact(token, s, items)

	pk := key.MustNewBTCPrivatekey()
	sig, err := operation.NewFactSignature(pk, fact, nil)
	t.NoError(err)

	fs := []operation.FactSign{operation.NewBaseFactSign(pk.Publickey(), sig)}

	tf, err := NewTransfers(fact, fs, "")
	t.NoError(err)

	err = tf.IsValid(nil)
	t.Contains(err.Error(), "receiver is same with sender")
}

func (t *testTransfers) TestOverSizeMemo() {
	s := MustAddress(util.UUID().String())
	r := MustAddress(util.UUID().String())

	token := util.UUID().Bytes()
	items := []TransferItem{NewTransferItem(r, NewAmount(10))}
	fact := NewTransfersFact(token, s, items)

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
	tf, err := NewTransfers(fact, fs, memo)
	t.NoError(err)

	err = tf.IsValid(nil)
	t.Contains(err.Error(), "memo over max size")
}

func TestTransfers(t *testing.T) {
	suite.Run(t, new(testTransfers))
}

func testTransfersEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestOperationEncode)

	t.enc = enc
	t.newObject = func() interface{} {
		s := MustAddress(util.UUID().String())
		r := MustAddress(util.UUID().String())

		token := util.UUID().Bytes()
		items := []TransferItem{NewTransferItem(r, NewAmount(10))}
		fact := NewTransfersFact(token, s, items)

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

		tf, err := NewTransfers(fact, fs, util.UUID().String())
		t.NoError(err)

		return tf
	}

	t.compare = func(a, b interface{}) {
		ta := a.(Transfers)
		tb := b.(Transfers)

		t.Equal(ta.Memo, tb.Memo)

		fact := ta.Fact().(TransfersFact)
		ufact := tb.Fact().(TransfersFact)

		t.True(fact.sender.Equal(ufact.sender))
		t.Equal(len(fact.Items()), len(ufact.Items()))

		for i := range fact.Items() {
			a := fact.Items()[i]
			b := ufact.Items()[i]

			t.True(a.receiver.Equal(b.receiver))
			t.Equal(a.amount, b.amount)
		}
	}

	return t
}

func TestTransfersEncodeJSON(t *testing.T) {
	suite.Run(t, testTransfersEncode(jsonenc.NewEncoder()))
}

func TestTransfersEncodeBSON(t *testing.T) {
	suite.Run(t, testTransfersEncode(bsonenc.NewEncoder()))
}
