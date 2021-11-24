//go:build mongodb
// +build mongodb

package digest

import (
	"context"
	"fmt"
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum-currency/currency"
)

type testDatabase struct {
	baseTest
}

func (t *testDatabase) TestInitialize() {
	st, err := NewDatabase(t.MongodbDatabase(), t.MongodbDatabase())
	t.NoError(err)

	newHeight := base.Height(33)
	t.NoError(st.SetLastBlock(newHeight))

	nst, err := NewDatabase(t.MongodbDatabase(), t.MongodbDatabase())
	t.NoError(err)
	t.NoError(nst.Initialize())

	h, found, err := loadLastBlock(st)
	t.NoError(err)
	t.True(found)

	t.Equal(newHeight, h)
}

func (t *testDatabase) TestOperationByAddress() {
	st, _ := t.Database()

	height := base.Height(3)
	confirmedAt := localtime.UTCNow()

	sender := currency.MustAddress(util.UUID().String())
	receiver0 := currency.MustAddress(util.UUID().String())
	receiver1 := currency.MustAddress(util.UUID().String())

	var hashes, hashes0, hashes1 []string

	{
		tf := t.newTransfer(sender, receiver0)
		doc, err := NewOperationDoc(tf, t.BSONEnc, height, confirmedAt, true, nil, 0)
		t.NoError(err)
		t.insertDoc(st, defaultColNameOperation, doc)

		hashes = append(hashes, tf.Fact().Hash().String())
		hashes0 = append(hashes0, tf.Fact().Hash().String())
	}

	{
		tf := t.newTransfer(sender, receiver1)
		doc, err := NewOperationDoc(tf, t.BSONEnc, height, confirmedAt, true, nil, 1)
		t.NoError(err)
		t.insertDoc(st, defaultColNameOperation, doc)

		hashes = append(hashes, tf.Fact().Hash().String())
		hashes1 = append(hashes1, tf.Fact().Hash().String())
	}

	{ // NOTE by sender
		var uhashes []string
		t.NoError(st.OperationsByAddress(
			sender,
			false,
			false,
			"",
			100,
			func(h valuehash.Hash, va OperationValue) (bool, error) {
				uhashes = append(uhashes, h.String())
				return true, nil
			},
		))

		t.Equal(hashes, uhashes)
	}

	{ // NOTE by receiver0
		var uhashes []string
		t.NoError(st.OperationsByAddress(
			receiver0,
			false,
			false,
			"",
			100,
			func(h valuehash.Hash, va OperationValue) (bool, error) {
				uhashes = append(uhashes, h.String())
				return true, nil
			},
		))

		t.Equal(hashes0, uhashes)
	}

	{ // NOTE by receiver1
		var uhashes []string
		t.NoError(st.OperationsByAddress(
			receiver1,
			false,
			false,
			"",
			100,
			func(h valuehash.Hash, va OperationValue) (bool, error) {
				uhashes = append(uhashes, h.String())
				return true, nil
			},
		))

		t.Equal(hashes1, uhashes)
	}
}

func (t *testDatabase) TestOperationByAddressOrderByHeight() {
	st, _ := t.Database()

	sender := currency.MustAddress(util.UUID().String())
	var hashes []string

	{
		height := base.Height(3)
		receiver := currency.MustAddress(util.UUID().String())
		{
			tf := t.newTransfer(sender, receiver)
			doc, err := NewOperationDoc(tf, t.BSONEnc, height, localtime.UTCNow(), true, nil, 0)
			t.NoError(err)
			t.insertDoc(st, defaultColNameOperation, doc)

			hashes = append(hashes, tf.Fact().Hash().String())
		}
	}

	{
		height := base.Height(4)
		receiver := currency.MustAddress(util.UUID().String())
		{
			tf := t.newTransfer(sender, receiver)
			doc, err := NewOperationDoc(tf, t.BSONEnc, height, localtime.UTCNow(), true, nil, 1)
			t.NoError(err)
			t.insertDoc(st, defaultColNameOperation, doc)

			hashes = append(hashes, tf.Fact().Hash().String())
		}
	}

	{ // height ascending
		var uhashes []string
		t.NoError(st.OperationsByAddress(
			sender,
			false,
			false,
			"",
			100,
			func(h valuehash.Hash, va OperationValue) (bool, error) {
				uhashes = append(uhashes, h.String())
				return true, nil
			},
		))

		t.Equal(hashes, uhashes)
	}

	{ // height descending
		var uhashes []string
		t.NoError(st.OperationsByAddress(
			sender,
			false,
			true,
			"",
			100,
			func(h valuehash.Hash, va OperationValue) (bool, error) {
				uhashes = append(uhashes, h.String())
				return true, nil
			},
		))

		rhashes := make([]string, len(hashes))
		copy(rhashes, hashes)
		for i, j := 0, len(rhashes)-1; i < j; i, j = i+1, j-1 {
			rhashes[i], rhashes[j] = rhashes[j], rhashes[i]
		}

		t.Equal(rhashes, uhashes)
	}
}

func (t *testDatabase) TestOperationByAddressOffset() {
	st, _ := t.Database()
	confirmedAt := localtime.UTCNow()

	sender := currency.MustAddress(util.UUID().String())
	var hashes []string
	hashesByHeight := map[string]base.Height{}

	for i := 0; i < 10; i++ {
		height := base.Height(i)
		tf := t.newTransfer(sender, currency.MustAddress(util.UUID().String()))
		doc, err := NewOperationDoc(tf, t.BSONEnc, height, confirmedAt, true, nil, 0)
		t.NoError(err)
		_ = t.insertDoc(st, defaultColNameOperation, doc)

		fh := tf.Fact().Hash().String()
		hashes = append(hashes, fh)
		hashesByHeight[fh] = height
	}

	{ // nil offset
		var uhashes []string
		t.NoError(st.OperationsByAddress(
			sender,
			false,
			false,
			"",
			100,
			func(h valuehash.Hash, va OperationValue) (bool, error) {
				uhashes = append(uhashes, h.String())
				return true, nil
			},
		))

		t.Equal(len(hashes), len(uhashes))
		t.Equal(hashes, uhashes)
	}

	{ // next of 3
		offset := buildOffset(hashesByHeight[hashes[3]], uint64(0))
		var uhashes []string
		t.NoError(st.OperationsByAddress(
			sender,
			false,
			false,
			offset,
			100,
			func(h valuehash.Hash, va OperationValue) (bool, error) {
				uhashes = append(uhashes, h.String())
				return true, nil
			},
		))

		t.Equal(len(hashes[4:]), len(uhashes))
		t.Equal(hashes[4:], uhashes)
	}

	{ // next of 9
		offset := buildOffset(hashesByHeight[hashes[9]], uint64(0))
		var uhashes []string
		t.NoError(st.OperationsByAddress(
			sender,
			false,
			false,
			offset,
			100,
			func(h valuehash.Hash, va OperationValue) (bool, error) {
				uhashes = append(uhashes, h.String())
				return true, nil
			},
		))

		t.Equal(0, len(uhashes))
		t.Empty(uhashes)
	}
}

func (t *testDatabase) TestOperationByAddressLimit() {
	st, _ := t.Database()

	sender := currency.MustAddress(util.UUID().String())
	var hashes []string

	for i := 0; i < 10; i++ {
		height := base.Height(i)
		tf := t.newTransfer(sender, currency.MustAddress(util.UUID().String()))
		doc, err := NewOperationDoc(tf, t.BSONEnc, height, localtime.UTCNow(), true, nil, uint64(i))
		t.NoError(err)
		_ = t.insertDoc(st, defaultColNameOperation, doc)

		hashes = append(hashes, tf.Fact().Hash().String())
	}

	var limit int64
	{ // limit 3
		limit = 3
		var uhashes []string
		t.NoError(st.OperationsByAddress(
			sender,
			false,
			false,
			"",
			limit,
			func(h valuehash.Hash, va OperationValue) (bool, error) {
				uhashes = append(uhashes, h.String())
				return true, nil
			},
		))

		t.Equal(int(limit), len(uhashes))
		t.Equal(hashes[:limit], uhashes)
	}

	{ // limit 3 with reverse
		rhashes := make([]string, len(hashes))
		copy(rhashes, hashes)
		for i, j := 0, len(rhashes)-1; i < j; i, j = i+1, j-1 {
			rhashes[i], rhashes[j] = rhashes[j], rhashes[i]
		}

		limit = 3
		var uhashes []string
		t.NoError(st.OperationsByAddress(
			sender,
			false,
			true,
			"",
			limit,
			func(h valuehash.Hash, va OperationValue) (bool, error) {
				uhashes = append(uhashes, h.String())
				return true, nil
			},
		))

		t.Equal(int(limit), len(uhashes))
		t.Equal(rhashes[:limit], uhashes)
	}

	{ // limit 9
		limit = 9
		var uhashes []string
		t.NoError(st.OperationsByAddress(
			sender,
			false,
			false,
			"",
			limit,
			func(h valuehash.Hash, va OperationValue) (bool, error) {
				uhashes = append(uhashes, h.String())
				return true, nil
			},
		))

		t.Equal(int(limit), len(uhashes))
		t.Equal(hashes[:limit], uhashes)
	}

	{ // over maxLimit
		limit = maxLimit + 10
		var uhashes []string
		t.NoError(st.OperationsByAddress(
			sender,
			false,
			false,
			"",
			limit,
			func(h valuehash.Hash, va OperationValue) (bool, error) {
				uhashes = append(uhashes, h.String())
				return true, nil
			},
		))

		t.Equal(len(hashes), len(uhashes))
		t.Equal(hashes, uhashes)
	}

	{ // negative limit -3; no limit
		limit = -3
		var uhashes []string
		t.NoError(st.OperationsByAddress(
			sender,
			false,
			false,
			"",
			limit,
			func(h valuehash.Hash, va OperationValue) (bool, error) {
				uhashes = append(uhashes, h.String())
				return true, nil
			},
		))

		t.Equal(len(hashes), len(uhashes))
		t.Equal(hashes, uhashes)
	}
}

func (t *testDatabase) TestOperationsFact() {
	st, _ := t.Database()
	height := base.Height(3)
	confirmedAt := localtime.UTCNow()

	var hashes []valuehash.Hash

	for i := 0; i < 3; i++ {
		tf := t.newTransfer(currency.MustAddress(util.UUID().String()), currency.MustAddress(util.UUID().String()))
		doc, err := NewOperationDoc(tf, t.BSONEnc, height, confirmedAt, true, nil, uint64(i))
		t.NoError(err)
		t.insertDoc(st, defaultColNameOperation, doc)

		hashes = append(hashes, tf.Fact().Hash())
	}

	for _, h := range hashes {
		va, exists, err := st.Operation(h, true)
		t.NoError(err)
		t.True(exists)
		t.True(h.Equal(va.Operation().Fact().Hash()))
	}

	unknown := valuehash.RandomSHA256()
	_, exists, err := st.Operation(unknown, true)
	t.NoError(err)
	t.False(exists)

	{ // not load
		_, exists, err := st.Operation(hashes[0], false)
		t.NoError(err)
		t.True(exists)
	}
}

func (t *testDatabase) TestClean() {
	st, _ := t.Database()

	sender := currency.MustAddress(util.UUID().String())

	lastHeight := base.Height(3)
	for height := base.GenesisHeight; height < lastHeight+1; height++ {
		tf := t.newTransfer(sender, currency.MustAddress(util.UUID().String()))
		doc, err := NewOperationDoc(tf, t.BSONEnc, height, localtime.UTCNow(), true, nil, uint64(height))
		t.NoError(err)
		t.insertDoc(st, defaultColNameOperation, doc)
	}

	t.NoError(st.SetLastBlock(lastHeight))

	t.NoError(st.Clean())

	h, found, err := loadLastBlock(st)
	t.NoError(err)
	t.True(found)
	t.Equal(base.NilHeight, h)

	var uhashes []string
	t.NoError(st.OperationsByAddress(
		sender,
		false,
		false,
		"",
		100,
		func(h valuehash.Hash, va OperationValue) (bool, error) {
			uhashes = append(uhashes, h.String())
			return true, nil
		},
	))

	t.Empty(uhashes)
}

func (t *testDatabase) TestCleanByHeight() {
	st, _ := t.Database()

	sender := currency.MustAddress(util.UUID().String())
	var hashes []string

	lastHeight := base.Height(10)
	for height := base.GenesisHeight; height < lastHeight+1; height++ {
		tf := t.newTransfer(sender, currency.MustAddress(util.UUID().String()))
		doc, err := NewOperationDoc(tf, t.BSONEnc, height, localtime.UTCNow(), true, nil, uint64(height))
		t.NoError(err)
		t.insertDoc(st, defaultColNameOperation, doc)

		hashes = append(hashes, tf.Fact().Hash().String())
	}

	t.NoError(st.SetLastBlock(lastHeight))

	height := base.Height(3)
	t.NoError(st.CleanByHeight(context.Background(), height))

	h, found, err := loadLastBlock(st)
	t.NoError(err)
	t.True(found)
	t.Equal(height-1, h)

	var uhashes []string
	t.NoError(st.OperationsByAddress(
		sender,
		false,
		false,
		"",
		100,
		func(h valuehash.Hash, va OperationValue) (bool, error) {
			uhashes = append(uhashes, h.String())
			return true, nil
		},
	))

	t.Equal(hashes[:3], uhashes)

	{ // NilHeight
		height := base.NilHeight
		t.NoError(st.CleanByHeight(context.Background(), height))

		h, found, err := loadLastBlock(st)
		t.NoError(err)
		t.True(found)
		t.Equal(base.NilHeight, h)
	}
}

func (t *testDatabase) TestAccountsWithBadState() {
	ac := t.newAccount()

	height := base.Height(33)
	st := t.newBalanceState(ac, height, currency.MustNewAmount(t.randomBig(), t.cid))

	_, err := NewAccountValue(st)
	t.Contains(err.Error(), "not state for currency.Account")
}

func (t *testDatabase) TestAccount() {
	st, _ := t.Database()

	height := base.Height(33)
	ac := t.newAccount()

	stA := t.newAccountState(ac, height)

	va, err := NewAccountValue(stA)
	t.NoError(err)

	docA, err := NewAccountDoc(va, t.BSONEnc)
	t.NoError(err)
	t.insertDoc(st, defaultColNameAccount, docA)

	am := currency.MustNewAmount(t.randomBig(), t.cid)
	stB := t.newBalanceState(ac, height, am)
	docB, err := NewBalanceDoc(stB, t.BSONEnc)
	t.NoError(err)
	t.insertDoc(st, defaultColNameBalance, docB)

	urs, found, err := st.Account(ac.Address())
	t.NoError(err)
	t.True(found)

	t.True(ac.Address().Equal(urs.ac.Address()))
	t.Equal(stA.Height(), urs.height)
	t.Equal(stA.PreviousHeight(), urs.previousHeight)
	t.Equal(1, len(urs.balance))
	t.compareAmount(am, urs.balance[0])
}

func (t *testDatabase) TestAccountBalanceUpdated() {
	st, _ := t.Database()

	ac := t.newAccount()

	height := base.Height(33)

	stA := t.newAccountState(ac, height)

	va, err := NewAccountValue(stA)
	t.NoError(err)

	docA, err := NewAccountDoc(va, t.BSONEnc)
	t.NoError(err)
	t.insertDoc(st, defaultColNameAccount, docA)

	var stB0, stB1 state.State
	{
		stB0 = t.newBalanceState(ac, height, currency.MustNewAmount(t.randomBig(), t.cid))
		_, err := currency.StateBalanceValue(stB0)
		t.NoError(err)
		docB, err := NewBalanceDoc(stB0, t.BSONEnc)
		t.NoError(err)
		t.insertDoc(st, defaultColNameBalance, docB)
	}

	lastAmount := currency.MustNewAmount(t.randomBig(), t.cid)
	{
		height = height + 3

		stB1 = t.newBalanceState(ac, height, lastAmount)
		docB, err := NewBalanceDoc(stB1, t.BSONEnc)
		t.NoError(err)
		t.insertDoc(st, defaultColNameBalance, docB)
	}

	urs, found, err := st.Account(ac.Address())
	t.NoError(err)
	t.True(found)

	t.True(ac.Address().Equal(urs.ac.Address()))
	t.Equal(stB1.Height(), urs.height)
	t.Equal(stB1.PreviousHeight(), urs.previousHeight)
	t.Equal(1, len(urs.balance))
	t.compareAmount(lastAmount, urs.balance[0])
}

func (t *testDatabase) TestAccountMultiCurrencies() {
	st, _ := t.Database()

	height := base.Height(33)
	ac := t.newAccount()

	stA := t.newAccountState(ac, height)

	va, err := NewAccountValue(stA)
	t.NoError(err)

	docA, err := NewAccountDoc(va, t.BSONEnc)
	t.NoError(err)
	t.insertDoc(st, defaultColNameAccount, docA)

	amB := currency.MustNewAmount(t.randomBig(), t.cid)
	stB := t.newBalanceState(ac, height, amB)
	docB, err := NewBalanceDoc(stB, t.BSONEnc)
	t.NoError(err)
	t.insertDoc(st, defaultColNameBalance, docB)

	cidC := currency.CurrencyID("EATME")
	amC := currency.MustNewAmount(t.randomBig(), cidC)
	stC := t.newBalanceState(ac, height, amC)
	docC, err := NewBalanceDoc(stC, t.BSONEnc)
	t.NoError(err)
	t.insertDoc(st, defaultColNameBalance, docC)

	urs, found, err := st.Account(ac.Address())
	t.NoError(err)
	t.True(found)

	t.Equal(2, len(urs.Balance()))
	balances := map[currency.CurrencyID]currency.Amount{}
	for i := range urs.Balance() {
		am := urs.Balance()[i]
		balances[am.Currency()] = am
	}

	amD := balances[t.cid]
	t.compareAmount(amB, amD)

	amE := balances[cidC]
	t.compareAmount(amC, amE)
}

func (t *testDatabase) TestOperations() {
	st, _ := t.Database()

	var hashes []string

	hashesByHeight := map[base.Height][]string{}
	for i := 0; i < 3; i++ {
		var hs []string
		height := base.Height(i)
		for j := 0; j < 3; j++ {
			tf := t.newTransfer(currency.MustAddress(util.UUID().String()), currency.MustAddress(util.UUID().String()))
			doc, err := NewOperationDoc(tf, t.BSONEnc, height, localtime.UTCNow(), true, nil, uint64(j))
			t.NoError(err)
			_ = t.insertDoc(st, defaultColNameOperation, doc)

			hashes = append(hashes, tf.Fact().Hash().String())
			hs = append(hs, tf.Fact().Hash().String())
		}

		hashesByHeight[height] = hs
	}

	{ // NOTE no offset

		reverse := false
		offset := ""
		filter, err := buildOperationsFilterByOffset(offset, reverse)
		t.NoError(err)

		var uhashes []string
		t.NoError(st.Operations(
			filter,
			false,
			reverse,
			100,
			func(h valuehash.Hash, va OperationValue) (bool, error) {
				uhashes = append(uhashes, h.String())
				return true, nil
			},
		))

		t.Equal(hashes, uhashes)
	}

	{ // NOTE offset
		reverse := false
		offset := buildOffset(base.GenesisHeight, 1)
		filter, err := buildOperationsFilterByOffset(offset, reverse)
		t.NoError(err)

		var uhashes []string
		t.NoError(st.Operations(
			filter,
			false,
			reverse,
			100,
			func(h valuehash.Hash, va OperationValue) (bool, error) {
				uhashes = append(uhashes, h.String())
				return true, nil
			},
		))

		t.Equal(hashes[2:], uhashes)
	}

	{ // NOTE over offset
		reverse := false
		offset := buildOffset(base.Height(4), 1)
		filter, err := buildOperationsFilterByOffset(offset, reverse)
		t.NoError(err)

		var uhashes []string
		t.NoError(st.Operations(
			filter,
			false,
			reverse,
			100,
			func(h valuehash.Hash, va OperationValue) (bool, error) {
				uhashes = append(uhashes, h.String())
				return true, nil
			},
		))

		t.Empty(uhashes)
	}

	{ // NOTE no offset by height
		height := base.Height(1)
		reverse := false
		filter, err := buildOperationsByHeightFilterByOffset(height, "", reverse)
		t.NoError(err)

		var uhashes []string
		t.NoError(st.Operations(
			filter,
			false,
			reverse,
			100,
			func(h valuehash.Hash, va OperationValue) (bool, error) {
				uhashes = append(uhashes, h.String())
				return true, nil
			},
		))

		t.Equal(hashesByHeight[height], uhashes)
	}

	{ // NOTE offset by height
		height := base.Height(1)
		reverse := false
		filter, err := buildOperationsByHeightFilterByOffset(height, fmt.Sprintf("%d", 0), reverse)
		t.NoError(err)

		var uhashes []string
		t.NoError(st.Operations(
			filter,
			false,
			reverse,
			100,
			func(h valuehash.Hash, va OperationValue) (bool, error) {
				uhashes = append(uhashes, h.String())
				return true, nil
			},
		))

		t.Equal(hashesByHeight[height][1:], uhashes)
	}
}

func TestDatabase(t *testing.T) {
	suite.Run(t, new(testDatabase))
}
