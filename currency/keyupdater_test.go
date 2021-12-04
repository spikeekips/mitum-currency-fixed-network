package currency

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
)

type testKeyUpdater struct {
	baseTest
}

func (t *testKeyUpdater) TestNew() {
	spk := key.MustNewBTCPrivatekey()
	skey, err := NewBaseAccountKey(spk.Publickey(), 100)
	t.NoError(err)
	skeys, err := NewBaseAccountKeys([]AccountKey{skey}, 100)
	t.NoError(err)
	sender, err := NewAddressFromKeys(skeys)
	t.NoError(err)

	npk := key.MustNewBTCPrivatekey()
	nkey, err := NewBaseAccountKey(npk.Publickey(), 100)
	t.NoError(err)
	nkeys, err := NewBaseAccountKeys([]AccountKey{nkey}, 100)
	t.NoError(err)

	token := util.UUID().Bytes()

	fact := NewKeyUpdaterFact(token, sender, nkeys, t.cid)
	sig, err := base.NewFactSignature(spk, fact, nil)
	t.NoError(err)
	fs := []base.FactSign{base.NewBaseFactSign(spk.Publickey(), sig)}

	op, err := NewKeyUpdater(fact, fs, "")
	t.NoError(err)

	t.NoError(op.IsValid(nil))

	t.Implements((*base.Fact)(nil), op.Fact())
	t.Implements((*operation.Operation)(nil), op)
}

func TestKeyUpdater(t *testing.T) {
	suite.Run(t, new(testKeyUpdater))
}

func testKeyUpdaterEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestOperationEncode)

	t.enc = enc
	t.newObject = func() interface{} {
		spk := key.MustNewBTCPrivatekey()
		skey, err := NewBaseAccountKey(spk.Publickey(), 100)
		t.NoError(err)
		skeys, err := NewBaseAccountKeys([]AccountKey{skey}, 100)
		t.NoError(err)
		sender, err := NewAddressFromKeys(skeys)
		t.NoError(err)

		npk := key.MustNewBTCPrivatekey()
		nkey, err := NewBaseAccountKey(npk.Publickey(), 100)
		t.NoError(err)
		nkeys, err := NewBaseAccountKeys([]AccountKey{nkey}, 100)
		t.NoError(err)

		token := util.UUID().Bytes()

		fact := NewKeyUpdaterFact(token, sender, nkeys, CurrencyID("SEEME"))
		sig, err := base.NewFactSignature(spk, fact, nil)
		t.NoError(err)
		fs := []base.FactSign{base.NewBaseFactSign(spk.Publickey(), sig)}

		op, err := NewKeyUpdater(fact, fs, "")
		t.NoError(err)

		t.NoError(op.IsValid(nil))

		return op
	}

	t.compare = func(a, b interface{}) {
		ca := a.(KeyUpdater)
		cb := b.(KeyUpdater)

		t.Equal(ca.Memo, cb.Memo)

		fact := ca.Fact().(KeyUpdaterFact)
		ufact := cb.Fact().(KeyUpdaterFact)

		t.True(fact.target.Equal(ufact.target))
		t.True(fact.Keys().Equal(ufact.Keys()))
		t.Equal(fact.currency, ufact.currency)
	}

	return t
}

func TestKeyUpdaterEncodeJSON(t *testing.T) {
	suite.Run(t, testKeyUpdaterEncode(jsonenc.NewEncoder()))
}

func TestKeyUpdaterEncodeBSON(t *testing.T) {
	suite.Run(t, testKeyUpdaterEncode(bsonenc.NewEncoder()))
}
