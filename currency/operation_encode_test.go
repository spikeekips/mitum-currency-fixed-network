package currency

import (
	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/localtime"
)

type baseTestEncode struct {
	suite.Suite

	enc       encoder.Encoder
	encs      *encoder.Encoders
	newObject func() interface{}
	encode    func(encoder.Encoder, interface{}) ([]byte, error)
	decode    func(encoder.Encoder, []byte) (interface{}, error)
	compare   func(interface{}, interface{})
}

func (t *baseTestEncode) SetupSuite() {
	t.encs = encoder.NewEncoders()
	t.encs.AddEncoder(t.enc)

	t.encs.TestAddHinter(key.BTCPublickeyHinter)
	t.encs.TestAddHinter(base.StringAddress(""))
	t.encs.TestAddHinter(Address(""))
	t.encs.TestAddHinter(base.BaseFactSign{})
	t.encs.TestAddHinter(Key{})
	t.encs.TestAddHinter(Keys{})
	t.encs.TestAddHinter(TransfersFact{})
	t.encs.TestAddHinter(TransfersHinter)
	t.encs.TestAddHinter(CreateAccountsFact{})
	t.encs.TestAddHinter(CreateAccountsHinter)
	t.encs.TestAddHinter(KeyUpdaterFact{})
	t.encs.TestAddHinter(KeyUpdaterHinter)
	t.encs.TestAddHinter(FeeOperationFact{})
	t.encs.TestAddHinter(FeeOperation{})
	t.encs.TestAddHinter(Account{})
	t.encs.TestAddHinter(GenesisCurrenciesFact{})
	t.encs.TestAddHinter(GenesisCurrenciesHinter)
	t.encs.TestAddHinter(Amount{})
	t.encs.TestAddHinter(CreateAccountsItemMultiAmountsHinter)
	t.encs.TestAddHinter(CreateAccountsItemSingleAmountHinter)
	t.encs.TestAddHinter(TransfersItemMultiAmountsHinter)
	t.encs.TestAddHinter(TransfersItemSingleAmountHinter)
	t.encs.TestAddHinter(CurrencyRegisterFact{})
	t.encs.TestAddHinter(CurrencyRegisterHinter)
	t.encs.TestAddHinter(CurrencyDesign{})
	t.encs.TestAddHinter(NilFeeer{})
	t.encs.TestAddHinter(FixedFeeer{})
	t.encs.TestAddHinter(RatioFeeer{})
	t.encs.TestAddHinter(CurrencyPolicyUpdaterFact{})
	t.encs.TestAddHinter(CurrencyPolicyUpdaterHinter)
	t.encs.TestAddHinter(CurrencyPolicy{})
	t.encs.TestAddHinter(SuffrageInflationFact{})
	t.encs.TestAddHinter(SuffrageInflationHinter)
}

func (t *baseTestEncode) TestEncode() {
	i := t.newObject()

	var err error

	var b []byte
	if t.encode != nil {
		b, err = t.encode(t.enc, i)
		t.NoError(err)
	} else {
		b, err = t.enc.Marshal(i)
		t.NoError(err)
	}

	var v interface{}
	if t.decode != nil {
		v, err = t.decode(t.enc, b)
		t.NoError(err)
	} else {
		v, err = t.enc.Decode(b)
		t.NoError(err)
	}

	t.compare(i, v)
}

func (t *baseTestEncode) compareCurrencyDesign(a, b CurrencyDesign) {
	t.True(a.Amount.Equal(b.Amount))
	t.True(a.GenesisAccount().Equal(a.GenesisAccount()))
	t.Equal(a.Policy(), b.Policy())
	t.True(a.Aggregate().Equal(b.Aggregate()))
}

type baseTestOperationEncode struct {
	baseTestEncode
}

func (t *baseTestOperationEncode) TestEncode() {
	i := t.newObject()
	op, ok := i.(operation.Operation)
	t.True(ok)

	b, err := t.enc.Marshal(op)
	t.NoError(err)

	hinter, err := t.enc.Decode(b)
	t.NoError(err)

	uop, ok := hinter.(operation.Operation)
	t.True(ok)

	fact := op.Fact().(operation.OperationFact)
	ufact := uop.Fact().(operation.OperationFact)
	t.True(fact.Hash().Equal(ufact.Hash()))
	t.True(fact.Hint().Equal(ufact.Hint()))
	t.Equal(fact.Token(), ufact.Token())

	t.True(op.Hash().Equal(uop.Hash()))

	t.Equal(len(op.Signs()), len(uop.Signs()))
	for i := range op.Signs() {
		a := op.Signs()[i]
		b := uop.Signs()[i]

		t.True(a.Signer().Equal(b.Signer()))
		t.Equal(a.Signature(), b.Signature())
		t.True(localtime.Equal(a.SignedAt(), b.SignedAt()))
	}

	t.compare(op, uop)
}
