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

type testTransfers struct {
	suite.Suite
}

func (t *testTransfers) TestNew() {
	s := MustAddress(util.UUID().String())
	r := MustAddress(util.UUID().String())

	token := util.UUID().Bytes()
	am := []Amount{NewAmount(NewBig(11), CurrencyID("SHOWME"))}
	items := []TransfersItem{NewTransfersItemMultiAmounts(r, am)}
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

func (t *testTransfers) TestDuplicatedReceivers() {
	s := MustAddress(util.UUID().String())
	r := MustAddress(util.UUID().String())

	token := util.UUID().Bytes()

	ams := []Amount{NewAmount(NewBig(11), CurrencyID("SHOWME"))}

	items := []TransfersItem{
		NewTransfersItemMultiAmounts(r, ams),
		NewTransfersItemMultiAmounts(r, ams),
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

	ams := []Amount{NewAmount(NewBig(11), CurrencyID("SHOWME"))}
	items := []TransfersItem{
		NewTransfersItemMultiAmounts(r, ams),
		NewTransfersItemMultiAmounts(s, ams),
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
	ams := []Amount{NewAmount(NewBig(11), CurrencyID("SHOWME"))}
	items := []TransfersItem{NewTransfersItemMultiAmounts(r, ams)}
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
