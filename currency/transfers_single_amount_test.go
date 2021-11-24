package currency

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/stretchr/testify/suite"
)

type testTransfersItemSingleAmount struct {
	suite.Suite
}

func (t *testTransfersItemSingleAmount) TestNew() {
	s := MustAddress(util.UUID().String())
	r := MustAddress(util.UUID().String())

	token := util.UUID().Bytes()
	am := NewAmount(NewBig(11), CurrencyID("SHOWME"))
	items := []TransfersItem{NewTransfersItemSingleAmount(r, am)}
	fact := NewTransfersFact(token, s, items)

	var fs []base.FactSign

	for _, pk := range []key.Privatekey{
		key.MustNewBTCPrivatekey(),
		key.MustNewBTCPrivatekey(),
		key.MustNewBTCPrivatekey(),
	} {
		sig, err := base.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, base.NewBaseFactSign(pk.Publickey(), sig))
	}

	tf, err := NewTransfers(fact, fs, "")
	t.NoError(err)

	t.NoError(tf.IsValid(nil))

	t.Implements((*base.Fact)(nil), tf.Fact())
	t.Implements((*operation.Operation)(nil), tf)
}

func (t *testTransfersItemSingleAmount) TestZeroBig() {
	s := MustAddress(util.UUID().String())
	r := MustAddress(util.UUID().String())

	token := util.UUID().Bytes()
	am := NewAmount(NewBig(0), CurrencyID("SHOWME"))
	items := []TransfersItem{NewTransfersItemSingleAmount(r, am)}

	err := items[0].IsValid(nil)
	t.Contains(err.Error(), "amount should be over zero")

	fact := NewTransfersFact(token, s, items)

	pk := key.MustNewBTCPrivatekey()
	sig, err := base.NewFactSignature(pk, fact, nil)
	t.NoError(err)

	fs := []base.FactSign{base.NewBaseFactSign(pk.Publickey(), sig)}

	tf, err := NewTransfers(fact, fs, "")
	t.NoError(err)

	err = tf.IsValid(nil)
	t.Contains(err.Error(), "amount should be over zero")
}

func (t *testTransfersItemSingleAmount) TestOverMaxAmounts() {
	s := MustAddress(util.UUID().String())
	r := MustAddress(util.UUID().String())

	token := util.UUID().Bytes()

	ams := []Amount{
		NewAmount(NewBig(11), CurrencyID("FINDME0")),
		NewAmount(NewBig(22), CurrencyID("FINDME1")),
	}

	item := NewTransfersItemSingleAmount(r, ams[0])
	item.amounts = ams

	items := []TransfersItem{item}

	err := items[0].IsValid(nil)
	t.Contains(err.Error(), "only one amount allowed")

	fact := NewTransfersFact(token, s, items)

	pk := key.MustNewBTCPrivatekey()
	sig, err := base.NewFactSignature(pk, fact, nil)
	t.NoError(err)

	fs := []base.FactSign{base.NewBaseFactSign(pk.Publickey(), sig)}

	tf, err := NewTransfers(fact, fs, "")
	t.NoError(err)

	err = tf.IsValid(nil)
	t.Contains(err.Error(), "only one amount allowed")
}

func (t *testTransfersItemSingleAmount) TestEmptyAmounts() {
	s := MustAddress(util.UUID().String())
	r := MustAddress(util.UUID().String())

	token := util.UUID().Bytes()
	item := NewTransfersItemSingleAmount(r, NewAmount(NewBig(11), CurrencyID("FINDME0")))
	item.amounts = nil

	items := []TransfersItem{item}

	err := items[0].IsValid(nil)
	t.Contains(err.Error(), "empty amounts")

	fact := NewTransfersFact(token, s, items)

	pk := key.MustNewBTCPrivatekey()
	sig, err := base.NewFactSignature(pk, fact, nil)
	t.NoError(err)

	fs := []base.FactSign{base.NewBaseFactSign(pk.Publickey(), sig)}

	tf, err := NewTransfers(fact, fs, "")
	t.NoError(err)

	err = tf.IsValid(nil)
	t.Contains(err.Error(), "empty amounts")
}

func TestTransfersItemSingleAmount(t *testing.T) {
	suite.Run(t, new(testTransfersItemSingleAmount))
}

func testTransfersItemSingleAmountEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestOperationEncode)

	t.enc = enc
	t.newObject = func() interface{} {
		s := MustAddress(util.UUID().String())
		r := MustAddress(util.UUID().String())

		token := util.UUID().Bytes()
		items := []TransfersItem{
			NewTransfersItemSingleAmount(r, NewAmount(NewBig(33), CurrencyID("SHOWME"))),
			NewTransfersItemSingleAmount(r, NewAmount(NewBig(44), CurrencyID("FINDME"))),
		}
		fact := NewTransfersFact(token, s, items)

		var fs []base.FactSign

		for _, pk := range []key.Privatekey{
			key.MustNewBTCPrivatekey(),
			key.MustNewBTCPrivatekey(),
			key.MustNewBTCPrivatekey(),
		} {
			sig, err := base.NewFactSignature(pk, fact, nil)
			t.NoError(err)

			fs = append(fs, base.NewBaseFactSign(pk.Publickey(), sig))
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

			t.True(a.Receiver().Equal(b.Receiver()))
			for j := range a.Amounts() {
				aam := a.Amounts()[j]
				bam := b.Amounts()[j]
				t.True(aam.Equal(bam))
			}
		}
	}

	return t
}

func TestTransfersItemSingleAmountEncodeJSON(t *testing.T) {
	suite.Run(t, testTransfersItemSingleAmountEncode(jsonenc.NewEncoder()))
}

func TestTransfersItemSingleAmountEncodeBSON(t *testing.T) {
	suite.Run(t, testTransfersItemSingleAmountEncode(bsonenc.NewEncoder()))
}
