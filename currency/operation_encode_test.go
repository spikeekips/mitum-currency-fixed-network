package currency

import (
	"github.com/stretchr/testify/suite"

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

	t.encs.AddHinter(key.BTCPublickeyHinter)
	t.encs.AddHinter(Address(""))
	t.encs.AddHinter(operation.BaseFactSign{})
	t.encs.AddHinter(Key{})
	t.encs.AddHinter(Keys{})
	t.encs.AddHinter(TransfersFact{})
	t.encs.AddHinter(Transfers{})
	t.encs.AddHinter(CreateAccountsFact{})
	t.encs.AddHinter(CreateAccounts{})
	t.encs.AddHinter(KeyUpdaterFact{})
	t.encs.AddHinter(KeyUpdater{})
	t.encs.AddHinter(FeeOperationFact{})
	t.encs.AddHinter(FeeOperation{})
	t.encs.AddHinter(Account{})
	t.encs.AddHinter(GenesisCurrenciesFact{})
	t.encs.AddHinter(GenesisCurrencies{})
	t.encs.AddHinter(Amount{})
	t.encs.AddHinter(CreateAccountsItemMultiAmountsHinter)
	t.encs.AddHinter(CreateAccountsItemSingleAmountHinter)
	t.encs.AddHinter(TransfersItemMultiAmountsHinter)
	t.encs.AddHinter(TransfersItemSingleAmountHinter)
	t.encs.AddHinter(CurrencyRegisterFact{})
	t.encs.AddHinter(CurrencyRegister{})
	t.encs.AddHinter(CurrencyDesign{})
	t.encs.AddHinter(NilFeeer{})
	t.encs.AddHinter(FixedFeeer{})
	t.encs.AddHinter(RatioFeeer{})
	t.encs.AddHinter(CurrencyPolicyUpdaterFact{})
	t.encs.AddHinter(CurrencyPolicyUpdater{})
	t.encs.AddHinter(CurrencyPolicy{})
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
		v, err = t.enc.DecodeByHint(b)
		t.NoError(err)
	}

	t.compare(i, v)
}

func (t *baseTestEncode) compareCurrencyDesign(a, b CurrencyDesign) {
	t.True(a.Amount.Equal(b.Amount))
	t.True(a.GenesisAccount().Equal(a.GenesisAccount()))
	t.Equal(a.Policy(), b.Policy())
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

	hinter, err := t.enc.DecodeByHint(b)
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
