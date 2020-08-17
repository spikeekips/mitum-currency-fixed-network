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

type testCreateAccountOperation struct {
	baseTestOperationProcessor
}

func (t *testCreateAccountOperation) newOperation(sender base.Address, amount Amount, keys Keys, pks []key.Privatekey) CreateAccount {
	token := util.UUID().Bytes()
	fact := NewCreateAccountFact(token, sender, keys, amount)

	var fs []operation.FactSign
	for _, pk := range pks {
		sig, err := operation.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
	}

	ca, err := NewCreateAccount(fact, fs, "")
	t.NoError(err)

	t.NoError(ca.IsValid(nil))

	return ca
}

func (t *testCreateAccountOperation) TestSufficientBalance() {
	saBalance := NewAmount(33)
	sa := t.newAccount(true, saBalance)
	na := t.newAccount(false, NilAmount)

	amount := NewAmount(3)
	ca := t.newOperation(sa.Address, amount, na.Keys(), sa.Privs())

	err := t.opr.Process(ca)
	t.NoError(err)

	// checking value
	sstate, found, err := t.pool.Get(StateKeyBalance(sa.Address))
	t.NoError(err)
	t.True(found)
	t.NotNil(sstate)

	rstateBalance, found, err := t.pool.Get(StateKeyBalance(na.Address))
	t.NoError(err)
	t.True(found)
	t.NotNil(rstateBalance)

	t.Equal(sstate.Value().Interface(), saBalance.Sub(amount).String())
	t.Equal(rstateBalance.Value().Interface(), amount.String())

	rstate, found, err := t.pool.Get(StateKeyKeys(na.Address))
	t.NoError(err)
	t.True(found)
	t.NotNil(rstate)

	ukeys := rstate.Value().Interface().(Keys)
	t.Equal(len(na.Keys().Keys()), len(ukeys.Keys()))
	t.Equal(na.Keys().Threshold(), ukeys.Threshold())
	for i := range na.Keys().Keys() {
		t.Equal(na.Keys().Keys()[i].Weight(), ukeys.Keys()[i].Weight())
		t.True(na.Keys().Keys()[i].Key().Equal(ukeys.Keys()[i].Key()))
	}
}

func (t *testCreateAccountOperation) TestSenderKeysNotExist() {
	sa := t.newAccount(false, NilAmount)
	na := t.newAccount(false, NilAmount)

	amount := NewAmount(10)
	ca := t.newOperation(sa.Address, amount, na.Keys(), sa.Privs())

	err := t.opr.Process(ca)

	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "does not exist")
}

func (t *testCreateAccountOperation) TestSenderBalanceNotExist() {
	spk := key.MustNewBTCPrivatekey()

	skey := NewKey(spk.Publickey(), 100)
	skeys, _ := NewKeys([]Key{skey}, 100)

	sender, _ := NewAddressFromKeys([]Key{skey})
	_ = t.newStateKeys(sender, skeys)

	na := t.newAccount(false, NilAmount)

	amount := NewAmount(10)
	ca := t.newOperation(sender, amount, na.Keys(), []key.Privatekey{spk})

	err := t.opr.Process(ca)

	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "balance of sender does not exist")
}

func (t *testCreateAccountOperation) TestReceiverExists() {
	// set sender state
	senderBalance := NewAmount(33)
	sa := t.newAccount(true, senderBalance)
	na := t.newAccount(true, NewAmount(3))

	amount := NewAmount(10)

	ca := t.newOperation(sa.Address, amount, na.Keys(), sa.Privs())

	err := t.opr.Process(ca)
	t.Contains(err.Error(), "keys of target already exists")
}

func (t *testCreateAccountOperation) TestInsufficientBalance() {
	amount := NewAmount(10)
	senderBalance := amount.Sub(NewAmount(3))

	sa := t.newAccount(true, senderBalance)
	na := t.newAccount(false, NilAmount)

	ca := t.newOperation(sa.Address, amount, na.Keys(), sa.Privs())

	err := t.opr.Process(ca)
	t.Contains(err.Error(), "insufficient balance")
}

func (t *testCreateAccountOperation) TestSameSenders() {
	sa := t.newAccount(true, NewAmount(3))
	na0 := t.newAccount(false, NilAmount)

	ca0 := t.newOperation(sa.Address, NewAmount(1), na0.Keys(), sa.Privs())
	t.NoError(t.opr.Process(ca0))

	na1 := t.newAccount(false, NilAmount)
	ca1 := t.newOperation(sa.Address, NewAmount(1), na1.Keys(), sa.Privs())

	err := t.opr.Process(ca1)
	t.Contains(err.Error(), "violates only one sender")
}

func TestCreateAccountOperation(t *testing.T) {
	suite.Run(t, new(testCreateAccountOperation))
}
