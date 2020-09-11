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
	t.encs.AddHinter(GenesisAccountFact{})
	t.encs.AddHinter(GenesisAccount{})
	t.encs.AddHinter(KeyUpdaterFact{})
	t.encs.AddHinter(KeyUpdater{})
	t.encs.AddHinter(FeeOperationFact{})
	t.encs.AddHinter(FeeOperation{})
	t.encs.AddHinter(Account{})
}

func (t *baseTestEncode) TestEncode() {
	i := t.newObject()

	b, err := t.enc.Marshal(i)
	t.NoError(err)

	hinter, err := t.enc.DecodeByHint(b)
	t.NoError(err)

	t.compare(i, hinter)
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
		t.Equal(localtime.RFC3339(a.SignedAt()), localtime.RFC3339(b.SignedAt()))
	}

	t.compare(op, uop)
}
