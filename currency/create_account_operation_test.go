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

type testCreateAccountsOperation struct {
	baseTestOperationProcessor
}

func (t *testCreateAccountsOperation) newOperation(sender base.Address, items []CreateAccountItem, pks []key.Privatekey) CreateAccounts {
	token := util.UUID().Bytes()
	fact := NewCreateAccountsFact(token, sender, items)

	var fs []operation.FactSign
	for _, pk := range pks {
		sig, err := operation.NewFactSignature(pk, fact, nil)
		t.NoError(err)

		fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
	}

	ca, err := NewCreateAccounts(fact, fs, "")
	t.NoError(err)

	t.NoError(ca.IsValid(nil))

	return ca
}

func (t *testCreateAccountsOperation) TestSufficientBalance() {
	saBalance := NewAmount(33)
	sa, st := t.newAccount(true, saBalance)
	na, _ := t.newAccount(false, NilAmount)

	pool, opr := t.statepool(st)

	amount := NewAmount(3)
	items := []CreateAccountItem{NewCreateAccountItem(na.Keys(), amount)}
	ca := t.newOperation(sa.Address, items, sa.Privs())

	err := opr.Process(ca)
	t.NoError(err)

	// checking value
	var sb, ns, nb state.State
	for _, st := range pool.Updates() {
		if st.Key() == StateKeyBalance(sa.Address) {
			sb = st
		} else if st.Key() == StateKeyKeys(na.Address) {
			ns = st
		} else if st.Key() == StateKeyBalance(na.Address) {
			nb = st
		}
	}

	t.Equal(sb.Value().Interface(), saBalance.Sub(amount).String())
	t.Equal(nb.Value().Interface(), amount.String())

	ukeys := ns.Value().Interface().(Keys)
	t.Equal(len(na.Keys().Keys()), len(ukeys.Keys()))
	t.Equal(na.Keys().Threshold(), ukeys.Threshold())
	for i := range na.Keys().Keys() {
		t.Equal(na.Keys().Keys()[i].Weight(), ukeys.Keys()[i].Weight())
		t.True(na.Keys().Keys()[i].Key().Equal(ukeys.Keys()[i].Key()))
	}
}

func (t *testCreateAccountsOperation) TestSenderKeysNotExist() {
	sa, _ := t.newAccount(false, NilAmount)
	na, _ := t.newAccount(false, NilAmount)

	_, opr := t.statepool()

	amount := NewAmount(10)
	items := []CreateAccountItem{NewCreateAccountItem(na.Keys(), amount)}
	ca := t.newOperation(sa.Address, items, sa.Privs())

	err := opr.Process(ca)

	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "does not exist")
}

func (t *testCreateAccountsOperation) TestSenderBalanceNotExist() {
	spk := key.MustNewBTCPrivatekey()

	skey := NewKey(spk.Publickey(), 100)
	skeys, _ := NewKeys([]Key{skey}, 100)

	keys, err := NewKeys([]Key{skey}, 100)
	t.NoError(err)

	sender, _ := NewAddressFromKeys(keys)
	st := t.newStateKeys(sender, skeys)
	_, opr := t.statepool([]state.State{st})

	na, _ := t.newAccount(false, NilAmount)

	amount := NewAmount(10)
	items := []CreateAccountItem{NewCreateAccountItem(na.Keys(), amount)}
	ca := t.newOperation(sender, items, []key.Privatekey{spk})

	err = opr.Process(ca)

	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "balance of sender does not exist")
}

func (t *testCreateAccountsOperation) TestReceiverExists() {
	// set sender state
	senderBalance := NewAmount(33)
	sa, st0 := t.newAccount(true, senderBalance)
	na, st1 := t.newAccount(true, NewAmount(3))

	_, opr := t.statepool(st0, st1)

	amount := NewAmount(10)

	items := []CreateAccountItem{NewCreateAccountItem(na.Keys(), amount)}
	ca := t.newOperation(sa.Address, items, sa.Privs())

	err := opr.Process(ca)
	t.Contains(err.Error(), "keys of target already exists")
}

func (t *testCreateAccountsOperation) TestInsufficientBalance() {
	amount := NewAmount(10)
	senderBalance := amount.Sub(NewAmount(3))

	sa, st := t.newAccount(true, senderBalance)
	na, _ := t.newAccount(false, NilAmount)

	_, opr := t.statepool(st)

	items := []CreateAccountItem{NewCreateAccountItem(na.Keys(), amount)}
	ca := t.newOperation(sa.Address, items, sa.Privs())

	err := opr.Process(ca)

	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "insufficient balance")
}

func (t *testCreateAccountsOperation) TestInsufficientBalanceMultipleItems() {
	amount := NewAmount(10)
	senderBalance := amount.Sub(NewAmount(3))

	sa, st := t.newAccount(true, senderBalance)
	na0, _ := t.newAccount(false, NilAmount)
	na1, _ := t.newAccount(false, NilAmount)

	_, opr := t.statepool(st)

	items := []CreateAccountItem{
		NewCreateAccountItem(na0.Keys(), amount),
		NewCreateAccountItem(na1.Keys(), NewAmount(4)),
	}
	ca := t.newOperation(sa.Address, items, sa.Privs())

	err := opr.Process(ca)
	t.Contains(err.Error(), "insufficient balance")
}

func (t *testCreateAccountsOperation) TestSameSenders() {
	sa, st := t.newAccount(true, NewAmount(3))
	na0, _ := t.newAccount(false, NilAmount)

	_, opr := t.statepool(st)

	items := []CreateAccountItem{NewCreateAccountItem(na0.Keys(), NewAmount(1))}
	ca0 := t.newOperation(sa.Address, items, sa.Privs())
	t.NoError(opr.Process(ca0))

	na1, _ := t.newAccount(false, NilAmount)
	items = []CreateAccountItem{NewCreateAccountItem(na1.Keys(), NewAmount(1))}
	ca1 := t.newOperation(sa.Address, items, sa.Privs())

	err := opr.Process(ca1)
	t.Contains(err.Error(), "violates only one sender")
}

func TestCreateAccountsOperation(t *testing.T) {
	suite.Run(t, new(testCreateAccountsOperation))
}
