package currency

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
)

type testKeyUpdaterOperation struct {
	baseTestOperationProcessor
}

func (t *testKeyUpdaterOperation) newOperation(target base.Address, keys Keys, pks []key.Privatekey) KeyUpdater {
	token := util.UUID().Bytes()
	fact := NewKeyUpdaterFact(token, target, keys)

	var fs []operation.FactSign
	for _, pk := range pks {
		sig, err := operation.NewFactSignature(pk, fact, nil)
		if err != nil {
			panic(err)
		}

		fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
	}

	op, err := NewKeyUpdater(fact, fs, "")
	if err != nil {
		panic(err)
	}

	err = op.IsValid(nil)
	if err != nil {
		panic(err)
	}

	return op
}

func (t *testKeyUpdaterOperation) TestNew() {
	sa, st := t.newAccount(true, NewAmount(3))

	pool, opr := t.statepool(st)

	npk := key.MustNewBTCPrivatekey()
	nkey, err := NewKey(npk.Publickey(), 100)
	t.NoError(err)
	nkeys, err := NewKeys([]Key{nkey}, 100)
	t.NoError(err)

	op := t.newOperation(sa.Address, nkeys, sa.Privs())

	t.NoError(opr.Process(op))

	// checking value
	var ns state.State
	for _, st := range pool.Updates() {
		if st.Key() == StateKeyKeys(sa.Address) {
			ns = st
		}
	}

	ukeys := ns.Value().Interface().(Keys)
	t.True(nkeys.Equal(ukeys))
}

func (t *testKeyUpdaterOperation) TestTargetNotExist() {
	sa, _ := t.newAccount(false, NewAmount(3))

	_, opr := t.statepool()

	npk := key.MustNewBTCPrivatekey()
	nkey, err := NewKey(npk.Publickey(), 100)
	t.NoError(err)
	nkeys, err := NewKeys([]Key{nkey}, 100)
	t.NoError(err)

	op := t.newOperation(sa.Address, nkeys, sa.Privs())

	err = opr.Process(op)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "target keys does not exist")
}

func (t *testKeyUpdaterOperation) TestSameKeys() {
	sa, st := t.newAccount(true, NewAmount(3))

	_, opr := t.statepool(st)

	op := t.newOperation(sa.Address, sa.Keys(), sa.Privs())

	err := opr.Process(op)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "same Keys")
}

func TestKeyUpdaterOperation(t *testing.T) {
	suite.Run(t, new(testKeyUpdaterOperation))
}
