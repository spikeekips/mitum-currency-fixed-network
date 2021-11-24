package currency

import (
	"fmt"
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

type testTransfersItemMultiAmounts struct {
	suite.Suite
}

func (t *testTransfersItemMultiAmounts) TestNew() {
	s := MustAddress(util.UUID().String())
	r := MustAddress(util.UUID().String())

	token := util.UUID().Bytes()
	ams := []Amount{NewAmount(NewBig(11), CurrencyID("SHOWME"))}
	items := []TransfersItem{NewTransfersItemMultiAmounts(r, ams)}
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

func (t *testTransfersItemMultiAmounts) TestZeroBig() {
	s := MustAddress(util.UUID().String())
	r := MustAddress(util.UUID().String())

	token := util.UUID().Bytes()
	ams := []Amount{NewAmount(NewBig(0), CurrencyID("SHOWME"))}
	items := []TransfersItem{NewTransfersItemMultiAmounts(r, ams)}

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

func (t *testTransfersItemMultiAmounts) TestOverMaxAmounts() {
	s := MustAddress(util.UUID().String())
	r := MustAddress(util.UUID().String())

	token := util.UUID().Bytes()

	var ams []Amount
	for i := 0; i < maxCurenciesTransfersItemMultiAmounts+1; i++ {
		ams = append(ams, NewAmount(NewBig(11), CurrencyID(fmt.Sprintf("FINDME_%d", i))))
	}

	items := []TransfersItem{NewTransfersItemMultiAmounts(r, ams)}

	err := items[0].IsValid(nil)
	t.Contains(err.Error(), "amounts over allowed")

	fact := NewTransfersFact(token, s, items)

	pk := key.MustNewBTCPrivatekey()
	sig, err := base.NewFactSignature(pk, fact, nil)
	t.NoError(err)

	fs := []base.FactSign{base.NewBaseFactSign(pk.Publickey(), sig)}

	tf, err := NewTransfers(fact, fs, "")
	t.NoError(err)

	err = tf.IsValid(nil)
	t.Contains(err.Error(), "amounts over allowed")
}

func (t *testTransfersItemMultiAmounts) TestDuplicatedCurrency() {
	s := MustAddress(util.UUID().String())
	r := MustAddress(util.UUID().String())

	token := util.UUID().Bytes()

	ams := []Amount{
		NewAmount(NewBig(11), CurrencyID("FINDME")),
		NewAmount(NewBig(22), CurrencyID("FINDME")),
	}

	items := []TransfersItem{NewTransfersItemMultiAmounts(r, ams)}

	err := items[0].IsValid(nil)
	t.Contains(err.Error(), "duplicated currency found")

	fact := NewTransfersFact(token, s, items)

	pk := key.MustNewBTCPrivatekey()
	sig, err := base.NewFactSignature(pk, fact, nil)
	t.NoError(err)

	fs := []base.FactSign{base.NewBaseFactSign(pk.Publickey(), sig)}

	tf, err := NewTransfers(fact, fs, "")
	t.NoError(err)

	err = tf.IsValid(nil)
	t.Contains(err.Error(), "duplicated currency found")
}

func (t *testTransfersItemMultiAmounts) TestEmptyAmounts() {
	s := MustAddress(util.UUID().String())
	r := MustAddress(util.UUID().String())

	token := util.UUID().Bytes()
	items := []TransfersItem{NewTransfersItemMultiAmounts(r, nil)}

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

func TestTransfersItemMultiAmounts(t *testing.T) {
	suite.Run(t, new(testTransfersItemMultiAmounts))
}

func testTransfersItemMultiAmountsEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestOperationEncode)

	t.enc = enc
	t.newObject = func() interface{} {
		s := MustAddress(util.UUID().String())
		r := MustAddress(util.UUID().String())

		token := util.UUID().Bytes()
		items := []TransfersItem{
			NewTransfersItemMultiAmounts(r, []Amount{NewAmount(NewBig(33), CurrencyID("SHOWME"))}),
			NewTransfersItemMultiAmounts(r, []Amount{NewAmount(NewBig(44), CurrencyID("FINDME"))}),
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

func TestTransfersItemMultiAmountsEncodeJSON(t *testing.T) {
	suite.Run(t, testTransfersItemMultiAmountsEncode(jsonenc.NewEncoder()))
}

func TestTransfersItemMultiAmountsEncodeBSON(t *testing.T) {
	suite.Run(t, testTransfersItemMultiAmountsEncode(bsonenc.NewEncoder()))
}
