package currency

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

type testCreateAccountsOperation struct {
	baseTestOperationProcessor
}

func (t *testCreateAccountsOperation) processor(cp *CurrencyPool, pool *storage.Statepool) prprocessor.OperationProcessor {
	copr, err := NewOperationProcessor(nil).
		SetProcessor(CreateAccounts{}, NewCreateAccountsProcessor(cp))
	t.NoError(err)

	if pool == nil {
		return copr
	}

	return copr.New(pool)
}

func (t *testCreateAccountsOperation) newOperation(sender base.Address, items []CreateAccountsItem, pks []key.Privatekey) CreateAccounts {
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
	cid0 := CurrencyID("SHOWME")
	cid1 := CurrencyID("FINDME")

	balance := []Amount{
		NewAmount(NewBig(33), cid0),
		NewAmount(NewBig(33), cid1),
	}

	sa, st := t.newAccount(true, balance)
	na, _ := t.newAccount(false, nil)

	pool, _ := t.statepool(st)

	fee := NewBig(1)
	feeer := NewFixedFeeer(sa.Address, fee)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(cid0, NewBig(99), sa.Address, feeer)))
	t.NoError(cp.Set(t.newCurrencyDesignState(cid1, NewBig(99), sa.Address, feeer)))

	opr := t.processor(cp, pool)

	ams := []Amount{
		NewAmount(NewBig(11), cid0),
		NewAmount(NewBig(22), cid1),
	}

	items := []CreateAccountsItem{NewCreateAccountsItemMultiAmounts(na.Keys(), ams)}
	ca := t.newOperation(sa.Address, items, sa.Privs())

	t.NoError(opr.Process(ca))

	// checking value
	var ns state.State
	sb := map[CurrencyID]state.State{}
	nb := map[CurrencyID]state.State{}
	for _, stu := range pool.Updates() {
		if IsStateBalanceKey(stu.Key()) {
			st := stu.GetState()

			i, err := StateBalanceValue(st)
			t.NoError(err)

			if st.Key() == StateKeyBalance(sa.Address, i.Currency()) {
				sb[i.Currency()] = st
			} else if st.Key() == StateKeyBalance(na.Address, i.Currency()) {
				nb[i.Currency()] = st
			} else {
				continue
			}
		} else if stu.Key() == StateKeyAccount(na.Address) {
			ns = stu.GetState()
		}
	}

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

	t.Equal(len(ams), len(sb))
	t.Equal(len(ams), len(nb))

	t.NotNil(sb[cid0])
	t.NotNil(sb[cid1])

	t.NotNil(nb[cid0])
	t.NotNil(nb[cid1])

	sba0, _ := StateBalanceValue(sb[cid0])
	t.True(sba0.Big().Equal(balance[0].Big().Sub(ams[0].Big()).Sub(fee)))

	sba1, _ := StateBalanceValue(sb[cid1])
	t.True(sba1.Big().Equal(balance[1].Big().Sub(ams[1].Big()).Sub(fee)))

	t.Equal(fee, sb[cid0].(AmountState).Fee())
	t.Equal(fee, sb[cid1].(AmountState).Fee())

	nba0, _ := StateBalanceValue(nb[cid0])
	t.True(nba0.Big().Equal(ams[0].Big()))

	nba1, _ := StateBalanceValue(nb[cid1])
	t.True(nba1.Big().Equal(ams[1].Big()))
}

func (t *testCreateAccountsOperation) TestMultipleItemsWithFee() {
	cid0 := CurrencyID("SHOWME")
	cid1 := CurrencyID("FINDME")

	balance := []Amount{
		NewAmount(NewBig(33), cid0),
		NewAmount(NewBig(33), cid1),
	}

	sa, st := t.newAccount(true, balance)
	na0, _ := t.newAccount(false, nil)
	na1, _ := t.newAccount(false, nil)

	pool, _ := t.statepool(st)

	fee := NewBig(1)
	feeer := NewFixedFeeer(sa.Address, fee)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(cid0, NewBig(99), sa.Address, feeer)))
	t.NoError(cp.Set(t.newCurrencyDesignState(cid1, NewBig(99), sa.Address, feeer)))

	opr := t.processor(cp, pool)

	ams := []Amount{
		NewAmount(NewBig(11), cid0),
		NewAmount(NewBig(22), cid1),
	}

	items := []CreateAccountsItem{
		NewCreateAccountsItemMultiAmounts(na0.Keys(), []Amount{ams[0]}),
		NewCreateAccountsItemMultiAmounts(na1.Keys(), []Amount{ams[1]}),
	}
	ca := t.newOperation(sa.Address, items, sa.Privs())

	t.NoError(opr.Process(ca))

	var ns0, ns1 state.State
	sb := map[CurrencyID]state.State{}
	nb0 := map[CurrencyID]state.State{}
	nb1 := map[CurrencyID]state.State{}
	for _, stu := range pool.Updates() {
		if IsStateBalanceKey(stu.Key()) {
			st := stu.GetState()

			i, err := StateBalanceValue(st)
			t.NoError(err)

			if st.Key() == StateKeyBalance(sa.Address, i.Currency()) {
				sb[i.Currency()] = st
			} else if st.Key() == StateKeyBalance(na0.Address, i.Currency()) {
				nb0[i.Currency()] = st
			} else if st.Key() == StateKeyBalance(na1.Address, i.Currency()) {
				nb1[i.Currency()] = st
			} else {
				continue
			}

		} else if stu.Key() == StateKeyAccount(na0.Address) {
			ns0 = stu.GetState()
		} else if stu.Key() == StateKeyAccount(na1.Address) {
			ns1 = stu.GetState()
		}
	}

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

	t.Equal(len(ams), len(sb))
	t.Equal(len(items[0].Amounts()), len(nb0))
	t.Equal(len(items[1].Amounts()), len(nb1))

	sba0, _ := StateBalanceValue(sb[cid0])
	t.True(sba0.Big().Equal(balance[0].Big().Sub(ams[0].Big()).Sub(fee)))

	sba1, _ := StateBalanceValue(sb[cid1])
	t.True(sba1.Big().Equal(balance[1].Big().Sub(ams[1].Big()).Sub(fee)))

	t.Equal(fee, sb[cid0].(AmountState).Fee())
	t.Equal(fee, sb[cid1].(AmountState).Fee())

	nba0, _ := StateBalanceValue(nb0[cid0])
	t.True(nba0.Big().Equal(ams[0].Big()))

	nba1, _ := StateBalanceValue(nb1[cid1])
	t.True(nba1.Big().Equal(ams[1].Big()))
}

func (t *testCreateAccountsOperation) TestInSufficientBalanceWithMinBalance() {
	cid := CurrencyID("SHOWME")

	balance := []Amount{NewAmount(NewBig(33), cid)}

	sa, st := t.newAccount(true, balance)
	na, _ := t.newAccount(false, nil)

	pool, _ := t.statepool(st)

	fee := NewBig(4)
	feeer := NewFixedFeeer(sa.Address, fee)

	cp := NewCurrencyPool()

	minBalance := NewBig(100)
	de := NewCurrencyDesign(NewAmount(NewBig(99), cid), sa.Address, NewCurrencyPolicy(minBalance, feeer))

	st0, err := state.NewStateV0(StateKeyCurrencyDesign(cid), nil, base.NilHeight)
	t.NoError(err)

	nst, err := SetStateCurrencyDesignValue(st0, de)
	t.NoError(err)
	t.NoError(cp.Set(nst))

	opr := t.processor(cp, pool)

	ams := []Amount{NewAmount(NewBig(1), cid)}

	items := []CreateAccountsItem{NewCreateAccountsItemMultiAmounts(na.Keys(), ams)}
	ca := t.newOperation(sa.Address, items, sa.Privs())

	err = opr.Process(ca)

	var oper operation.ReasonError
	t.True(xerrors.As(err, &oper))
	t.Contains(err.Error(), "amount should be over minimum balance")
}

func (t *testCreateAccountsOperation) TestInSufficientBalanceWithFee() {
	cid := CurrencyID("SHOWME")

	balance := []Amount{NewAmount(NewBig(33), cid)}

	sa, st := t.newAccount(true, balance)
	na, _ := t.newAccount(false, nil)

	pool, _ := t.statepool(st)

	fee := NewBig(4)
	feeer := NewFixedFeeer(sa.Address, fee)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(cid, NewBig(99), sa.Address, feeer)))

	opr := t.processor(cp, pool)

	ams := []Amount{NewAmount(balance[0].Big(), cid)}

	items := []CreateAccountsItem{NewCreateAccountsItemMultiAmounts(na.Keys(), ams)}
	ca := t.newOperation(sa.Address, items, sa.Privs())

	err := opr.Process(ca)

	var oper operation.ReasonError
	t.True(xerrors.As(err, &oper))
	t.Contains(err.Error(), "insufficient balance")
}

func (t *testCreateAccountsOperation) TestUnknownCurrencyID() {
	cid := CurrencyID("SHOWME")

	balance := []Amount{NewAmount(NewBig(33), cid)}

	sa, st := t.newAccount(true, balance)
	na, _ := t.newAccount(false, nil)

	pool, _ := t.statepool(st)

	fee := NewBig(1)
	feeer := NewFixedFeeer(sa.Address, fee)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(CurrencyID("FINDME"), NewBig(99), sa.Address, feeer)))

	opr := t.processor(cp, pool)

	ams := []Amount{NewAmount(NewBig(1), cid)}

	items := []CreateAccountsItem{NewCreateAccountsItemMultiAmounts(na.Keys(), ams)}
	ca := t.newOperation(sa.Address, items, sa.Privs())

	err := opr.Process(ca)

	var oper operation.ReasonError
	t.True(xerrors.As(err, &oper))
	t.Contains(err.Error(), "unknown currency id found")
}

func (t *testCreateAccountsOperation) TestSenderKeysNotExist() {
	sa, _ := t.newAccount(false, nil)
	na, _ := t.newAccount(false, nil)

	cid := CurrencyID("FINDME")

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(cid, NewBig(99), sa.Address, NewNilFeeer())))

	_, opr := t.statepool()
	copr, err := opr.(*OperationProcessor).
		SetProcessor(CreateAccounts{}, NewCreateAccountsProcessor(cp))
	t.NoError(err)

	ams := []Amount{NewAmount(NewBig(33), cid)}

	items := []CreateAccountsItem{NewCreateAccountsItemMultiAmounts(na.Keys(), ams)}
	ca := t.newOperation(sa.Address, items, sa.Privs())

	err = copr.Process(ca)

	var oper operation.ReasonError
	t.True(xerrors.As(err, &oper))
	t.Contains(err.Error(), "does not exist")
}

func (t *testCreateAccountsOperation) TestEmptyCurrency() {
	cid0 := CurrencyID("SHOWME")
	cid1 := CurrencyID("FINDME")

	balance := []Amount{NewAmount(NewBig(33), cid0)}

	sa, st := t.newAccount(true, balance)
	na, _ := t.newAccount(false, nil)

	pool, _ := t.statepool(st)

	feeer := NewNilFeeer()

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(cid0, NewBig(99), sa.Address, feeer)))
	t.NoError(cp.Set(t.newCurrencyDesignState(cid1, NewBig(99), sa.Address, feeer)))

	opr := t.processor(cp, pool)

	ams := []Amount{NewAmount(NewBig(10), cid1)}

	items := []CreateAccountsItem{NewCreateAccountsItemMultiAmounts(na.Keys(), ams)}
	ca := t.newOperation(sa.Address, items, sa.Privs())

	err := opr.Process(ca)

	var operr operation.ReasonError
	t.True(xerrors.As(err, &operr))
	t.Contains(fmt.Sprintf("%+v", err), "currency of holder does not exist")
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
	copr, err := opr.(*OperationProcessor).
		SetProcessor(CreateAccounts{}, NewCreateAccountsProcessor(nil))
	t.NoError(err)

	na, _ := t.newAccount(false, nil)

	ams := []Amount{NewAmount(NewBig(10), CurrencyID("SHOWME"))}

	items := []CreateAccountsItem{NewCreateAccountsItemMultiAmounts(na.Keys(), ams)}
	ca := t.newOperation(sender, items, []key.Privatekey{spk})

	err = copr.Process(ca)

	var oper operation.ReasonError
	t.True(xerrors.As(err, &oper))
	t.Contains(err.Error(), "currency of holder does not exist")
}

func (t *testCreateAccountsOperation) TestReceiverExists() {
	// set sender state
	cid := CurrencyID("SHOWME")

	sa, st0 := t.newAccount(true, []Amount{NewAmount(NewBig(33), cid)})
	na, st1 := t.newAccount(true, []Amount{NewAmount(NewBig(3), cid)})

	pool, _ := t.statepool(st0, st1)
	feeer := NewFixedFeeer(sa.Address, ZeroBig)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(cid, NewBig(99), sa.Address, feeer)))

	opr := t.processor(cp, pool)

	items := []CreateAccountsItem{NewCreateAccountsItemMultiAmounts(na.Keys(), []Amount{NewAmount(NewBig(1), cid)})}
	ca := t.newOperation(sa.Address, items, sa.Privs())

	err := opr.Process(ca)
	t.Contains(err.Error(), "keys of target already exists")
}

func (t *testCreateAccountsOperation) TestInsufficientBalance() {
	cid := CurrencyID("SHOWME")

	big := NewBig(10)

	balance := []Amount{NewAmount(big.Sub(NewBig(3)), cid)}

	sa, st := t.newAccount(true, balance)
	na, _ := t.newAccount(false, nil)

	pool, _ := t.statepool(st)
	feeer := NewFixedFeeer(sa.Address, ZeroBig)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(cid, NewBig(99), sa.Address, feeer)))

	opr := t.processor(cp, pool)

	ams := []Amount{NewAmount(big, cid)}

	items := []CreateAccountsItem{NewCreateAccountsItemMultiAmounts(na.Keys(), ams)}
	ca := t.newOperation(sa.Address, items, sa.Privs())

	err := opr.Process(ca)

	var oper operation.ReasonError
	t.True(xerrors.As(err, &oper))
	t.Contains(err.Error(), "insufficient balance")
}

func (t *testCreateAccountsOperation) TestInsufficientBalanceMultipleItems() {
	cid := CurrencyID("SHOWME")

	big := NewBig(10)

	balance := []Amount{NewAmount(big.Sub(NewBig(3)), cid)}

	sa, st := t.newAccount(true, balance)
	na0, _ := t.newAccount(false, nil)
	na1, _ := t.newAccount(false, nil)

	pool, _ := t.statepool(st)
	feeer := NewFixedFeeer(sa.Address, ZeroBig)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(cid, NewBig(99), sa.Address, feeer)))

	opr := t.processor(cp, pool)

	items := []CreateAccountsItem{
		NewCreateAccountsItemMultiAmounts(na0.Keys(), []Amount{NewAmount(big, cid)}),
		NewCreateAccountsItemMultiAmounts(na1.Keys(), []Amount{NewAmount(NewBig(4), cid)}),
	}
	ca := t.newOperation(sa.Address, items, sa.Privs())

	err := opr.Process(ca)
	t.Contains(err.Error(), "insufficient balance")
}

func (t *testCreateAccountsOperation) TestSameSenders() {
	cid := CurrencyID("SHOWME")
	balance := []Amount{NewAmount(NewBig(33), cid)}

	sa, st := t.newAccount(true, balance)
	na0, _ := t.newAccount(false, nil)

	pool, _ := t.statepool(st)
	feeer := NewFixedFeeer(sa.Address, ZeroBig)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(cid, NewBig(99), sa.Address, feeer)))

	opr := t.processor(cp, pool)

	items := []CreateAccountsItem{NewCreateAccountsItemMultiAmounts(na0.Keys(), []Amount{NewAmount(NewBig(1), cid)})}
	ca0 := t.newOperation(sa.Address, items, sa.Privs())
	t.NoError(opr.Process(ca0))

	na1, _ := t.newAccount(false, nil)
	items = []CreateAccountsItem{NewCreateAccountsItemMultiAmounts(na1.Keys(), []Amount{NewAmount(NewBig(1), cid)})}
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
	cid := CurrencyID("SHOWME")
	balance := []Amount{NewAmount(NewBig(33), cid)}

	sa, st := t.newAccount(true, balance)
	na0, _ := t.newAccount(false, nil)

	pool, _ := t.statepool(st)

	feeer := NewFixedFeeer(sa.Address, ZeroBig)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(cid, NewBig(99), sa.Address, feeer)))

	opr := t.processor(cp, pool)

	// insert invalid operation, under threshold signing. It can not be counted
	// to sender checking.
	{
		na, _ := t.newAccount(false, nil)
		items := []CreateAccountsItem{NewCreateAccountsItemMultiAmounts(na.Keys(), []Amount{NewAmount(NewBig(1), cid)})}
		ca := t.newOperation(sa.Address, items, []key.Privatekey{key.MustNewBTCPrivatekey()})
		err := opr.Process(ca)

		var oper operation.ReasonError
		t.True(xerrors.As(err, &oper))
	}

	items := []CreateAccountsItem{NewCreateAccountsItemMultiAmounts(na0.Keys(), []Amount{NewAmount(NewBig(1), cid)})}
	ca0 := t.newOperation(sa.Address, items, sa.Privs())
	t.NoError(opr.Process(ca0))

	na1, _ := t.newAccount(false, nil)
	items = []CreateAccountsItem{NewCreateAccountsItemMultiAmounts(na1.Keys(), []Amount{NewAmount(NewBig(1), cid)})}
	ca1 := t.newOperation(sa.Address, items, sa.Privs())

	err := opr.Process(ca1)
	t.Contains(err.Error(), "violates only one sender")
}

func (t *testCreateAccountsOperation) TestSameAddress() {
	cid := CurrencyID("SHOWME")
	balance := []Amount{NewAmount(NewBig(33), cid)}

	sa, _ := t.newAccount(true, balance)
	na0, _ := t.newAccount(false, nil)

	it0 := NewCreateAccountsItemMultiAmounts(na0.Keys(), []Amount{NewAmount(NewBig(1), cid)})
	it1 := NewCreateAccountsItemMultiAmounts(na0.Keys(), []Amount{NewAmount(NewBig(1), cid)})
	items := []CreateAccountsItem{it0, it1}

	t.Panicsf(func() { t.newOperation(sa.Address, items, sa.Privs()) }, "duplicated acocunt Keys found")
}

func (t *testCreateAccountsOperation) TestSameAddressMultipleOperations() {
	cid := CurrencyID("SHOWME")
	balance := []Amount{NewAmount(NewBig(33), cid)}

	sa, sta := t.newAccount(true, balance)
	sb, stb := t.newAccount(true, balance)
	na0, _ := t.newAccount(false, nil)

	pool, _ := t.statepool(sta, stb)
	feeer := NewFixedFeeer(sa.Address, ZeroBig)

	cp := NewCurrencyPool()
	t.NoError(cp.Set(t.newCurrencyDesignState(cid, NewBig(99), sa.Address, feeer)))

	opr := t.processor(cp, pool)

	items := []CreateAccountsItem{NewCreateAccountsItemMultiAmounts(na0.Keys(), []Amount{NewAmount(NewBig(1), cid)})}
	ca0 := t.newOperation(sa.Address, items, sa.Privs())
	t.NoError(opr.Process(ca0))

	items = []CreateAccountsItem{NewCreateAccountsItemMultiAmounts(na0.Keys(), []Amount{NewAmount(NewBig(1), cid)})}
	ca1 := t.newOperation(sb.Address, items, sb.Privs())

	err := opr.Process(ca1)
	t.Contains(err.Error(), "new address already processed")
}

func TestCreateAccountsOperation(t *testing.T) {
	suite.Run(t, new(testCreateAccountsOperation))
}
