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
		if err != nil {
			panic(err)
		}

		fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
	}

	ca, err := NewCreateAccounts(fact, fs, "")
	if err != nil {
		panic(err)
	}

	err = ca.IsValid(nil)
	if err != nil {
		panic(err)
	}

	return ca
}

func (t *testCreateAccountsOperation) TestSufficientBalance() {
	saBalance := NewAmount(33)
	sa, st := t.newAccount(true, saBalance)
	na, _ := t.newAccount(false, NilAmount)

	pool, _ := t.statepool(st)

	fee := NewAmount(1)
	fa := NewFixedFeeAmount(fee)
	opr := NewOperationProcessor(fa, func() (base.Address, error) { return sa.Address, nil }).New(pool)

	amount := NewAmount(3)
	items := []CreateAccountItem{NewCreateAccountItem(na.Keys(), amount)}
	ca := t.newOperation(sa.Address, items, sa.Privs())

	err := opr.Process(ca)
	t.NoError(err)

	// checking value
	var sb, ns, nb state.State
	for _, st := range pool.Updates() {
		if st.Key() == StateKeyBalance(sa.Address) {
			sb = st.GetState()
		} else if st.Key() == StateKeyAccount(na.Address) {
			ns = st.GetState()
		} else if st.Key() == StateKeyBalance(na.Address) {
			nb = st.GetState()
		}
	}

	t.Equal(sb.Value().Interface(), saBalance.Sub(amount.Add(fee)).String())
	t.Equal(nb.Value().Interface(), amount.String())
	t.Equal(fee, sb.(AmountState).Fee())

	address, err := NewAddressFromKeys(na.Keys())
	t.NoError(err)
	uac := ns.Value().Interface().(Account)
	t.True(address.Equal(uac.Address()))

	ukeys := uac.Keys()

	t.Equal(len(na.Keys().Keys()), len(ukeys.Keys()))
	t.Equal(na.Keys().Threshold(), ukeys.Threshold())
	for i := range na.Keys().Keys() {
		t.Equal(na.Keys().Keys()[i].Weight(), ukeys.Keys()[i].Weight())
		t.True(na.Keys().Keys()[i].Key().Equal(ukeys.Keys()[i].Key()))
	}
}

func (t *testCreateAccountsOperation) TestMultipleItemsWithFee() {
	saBalance := NewAmount(33)
	sa, st := t.newAccount(true, saBalance)
	na0, _ := t.newAccount(false, NilAmount)
	na1, _ := t.newAccount(false, NilAmount)

	pool, _ := t.statepool(st)

	fee := NewAmount(1)
	fa := NewFixedFeeAmount(fee)
	opr := NewOperationProcessor(fa, func() (base.Address, error) { return sa.Address, nil }).New(pool)

	amount := NewAmount(3)
	items := []CreateAccountItem{
		NewCreateAccountItem(na0.Keys(), amount),
		NewCreateAccountItem(na1.Keys(), amount),
	}
	ca := t.newOperation(sa.Address, items, sa.Privs())

	err := opr.Process(ca)
	t.NoError(err)

	var sb, ns0, nb0, ns1, nb1 state.State
	for _, st := range pool.Updates() {
		if st.Key() == StateKeyBalance(sa.Address) {
			sb = st.GetState()
		} else if st.Key() == StateKeyAccount(na0.Address) {
			ns0 = st.GetState()
		} else if st.Key() == StateKeyBalance(na0.Address) {
			nb0 = st.GetState()
		} else if st.Key() == StateKeyAccount(na1.Address) {
			ns1 = st.GetState()
		} else if st.Key() == StateKeyBalance(na1.Address) {
			nb1 = st.GetState()
		}
	}

	totalFee := fee.MulInt64(2)
	totalAmount := amount.MulInt64(2).Add(totalFee)

	t.Equal(saBalance.Sub(totalAmount).String(), sb.Value().Interface())
	t.Equal(nb0.Value().Interface(), amount.String())
	t.Equal(nb1.Value().Interface(), amount.String())

	address0, err := NewAddressFromKeys(na0.Keys())
	t.NoError(err)
	uac0 := ns0.Value().Interface().(Account)
	t.True(address0.Equal(uac0.Address()))

	ukeys0 := uac0.Keys()

	t.Equal(len(na0.Keys().Keys()), len(ukeys0.Keys()))
	t.Equal(na0.Keys().Threshold(), ukeys0.Threshold())
	for i := range na0.Keys().Keys() {
		t.Equal(na0.Keys().Keys()[i].Weight(), ukeys0.Keys()[i].Weight())
		t.True(na0.Keys().Keys()[i].Key().Equal(ukeys0.Keys()[i].Key()))
	}

	address1, err := NewAddressFromKeys(na1.Keys())
	t.NoError(err)
	uac1 := ns1.Value().Interface().(Account)
	t.True(address1.Equal(uac1.Address()))

	ukeys1 := uac1.Keys()
	t.Equal(len(na1.Keys().Keys()), len(ukeys1.Keys()))
	t.Equal(na1.Keys().Threshold(), ukeys1.Threshold())
	for i := range na1.Keys().Keys() {
		t.Equal(na1.Keys().Keys()[i].Weight(), ukeys1.Keys()[i].Weight())
		t.True(na1.Keys().Keys()[i].Key().Equal(ukeys1.Keys()[i].Key()))
	}

	t.Equal(totalFee, sb.(AmountState).Fee())
}

func (t *testCreateAccountsOperation) TestInSufficientBalanceWithFee() {
	saBalance := NewAmount(33)
	sa, st := t.newAccount(true, saBalance)
	na, _ := t.newAccount(false, NilAmount)

	pool, _ := t.statepool(st)

	fee := NewAmount(4)
	fa := NewFixedFeeAmount(fee)
	opr := NewOperationProcessor(fa, func() (base.Address, error) { return sa.Address, nil }).New(pool)

	amount := NewAmount(30)
	items := []CreateAccountItem{NewCreateAccountItem(na.Keys(), amount)}
	ca := t.newOperation(sa.Address, items, sa.Privs())

	err := opr.Process(ca)
	t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	t.Contains(err.Error(), "insufficient balance")
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

	skey := t.newKey(spk.Publickey(), 100)
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

	pool, _ := t.statepool(st0, st1)
	opr := NewOperationProcessor(NewFixedFeeAmount(ZeroAmount), func() (base.Address, error) { return sa.Address, nil }).New(pool)

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

	pool, _ := t.statepool(st)
	opr := NewOperationProcessor(NewFixedFeeAmount(ZeroAmount), func() (base.Address, error) { return sa.Address, nil }).New(pool)

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

	pool, _ := t.statepool(st)
	opr := NewOperationProcessor(NewFixedFeeAmount(ZeroAmount), func() (base.Address, error) { return sa.Address, nil }).New(pool)

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

	pool, _ := t.statepool(st)
	opr := NewOperationProcessor(NewFixedFeeAmount(ZeroAmount), func() (base.Address, error) { return sa.Address, nil }).New(pool)

	items := []CreateAccountItem{NewCreateAccountItem(na0.Keys(), NewAmount(1))}
	ca0 := t.newOperation(sa.Address, items, sa.Privs())
	t.NoError(opr.Process(ca0))

	na1, _ := t.newAccount(false, NilAmount)
	items = []CreateAccountItem{NewCreateAccountItem(na1.Keys(), NewAmount(1))}
	ca1 := t.newOperation(sa.Address, items, sa.Privs())

	raddresses, err := ca1.Fact().(CreateAccountsFact).Addresses()
	t.NoError(err)
	t.Equal(2, len(raddresses))

	addresses := []base.Address{na1.Address, sa.Address}
	for i := range raddresses {
		t.True(addresses[i].Equal(raddresses[i]))
	}

	err = opr.Process(ca1)
	t.Contains(err.Error(), "violates only one sender")
}

func (t *testCreateAccountsOperation) TestSameSendersWithInvalidOperation() {
	sa, st := t.newAccount(true, NewAmount(3))
	na0, _ := t.newAccount(false, NilAmount)

	pool, _ := t.statepool(st)

	opr := NewOperationProcessor(NewFixedFeeAmount(ZeroAmount), func() (base.Address, error) { return sa.Address, nil }).New(pool)

	// insert invalid operation, under threshold signing. It can not be counted
	// to sender checking.
	{
		na, _ := t.newAccount(false, NilAmount)
		items := []CreateAccountItem{NewCreateAccountItem(na.Keys(), NewAmount(1))}
		ca := t.newOperation(sa.Address, items, []key.Privatekey{key.MustNewBTCPrivatekey()})
		err := opr.Process(ca)
		t.True(xerrors.Is(err, state.IgnoreOperationProcessingError))
	}

	items := []CreateAccountItem{NewCreateAccountItem(na0.Keys(), NewAmount(1))}
	ca0 := t.newOperation(sa.Address, items, sa.Privs())
	t.NoError(opr.Process(ca0))

	na1, _ := t.newAccount(false, NilAmount)
	items = []CreateAccountItem{NewCreateAccountItem(na1.Keys(), NewAmount(1))}
	ca1 := t.newOperation(sa.Address, items, sa.Privs())

	err := opr.Process(ca1)
	t.Contains(err.Error(), "violates only one sender")
}

func (t *testCreateAccountsOperation) TestSameAddress() {
	sa, _ := t.newAccount(true, NewAmount(3))
	na0, _ := t.newAccount(false, NilAmount)

	it0 := NewCreateAccountItem(na0.Keys(), NewAmount(1))
	it1 := NewCreateAccountItem(na0.Keys(), NewAmount(1))
	items := []CreateAccountItem{it0, it1}

	t.Panicsf(func() { t.newOperation(sa.Address, items, sa.Privs()) }, "duplicated acocunt Keys found")
}

func (t *testCreateAccountsOperation) TestSameAddressMultipleOperations() {
	sa, sta := t.newAccount(true, NewAmount(3))
	sb, stb := t.newAccount(true, NewAmount(3))
	na0, _ := t.newAccount(false, NilAmount)

	pool, _ := t.statepool(sta, stb)
	opr := NewOperationProcessor(NewFixedFeeAmount(ZeroAmount), func() (base.Address, error) { return sa.Address, nil }).New(pool)

	items := []CreateAccountItem{NewCreateAccountItem(na0.Keys(), NewAmount(1))}
	ca0 := t.newOperation(sa.Address, items, sa.Privs())
	t.NoError(opr.Process(ca0))

	items = []CreateAccountItem{NewCreateAccountItem(na0.Keys(), NewAmount(1))}
	ca1 := t.newOperation(sb.Address, items, sb.Privs())

	err := opr.Process(ca1)
	t.Contains(err.Error(), "new address already processed")
}

func TestCreateAccountsOperation(t *testing.T) {
	suite.Run(t, new(testCreateAccountsOperation))
}
