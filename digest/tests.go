//go:build test || mongodb
// +build test mongodb

package digest

import (
	"context"
	"crypto/rand"
	"math/big"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch"
	"github.com/spikeekips/mitum/storage"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum-currency/currency"
)

type baseTest struct { // nolint: unused
	suite.Suite
	isaac.StorageSupportTest
	networkID base.NetworkID
	cid       currency.CurrencyID
}

func (t *baseTest) SetupSuite() {
	t.DBType = "mongodb"
	t.StorageSupportTest.SetupSuite()

	for _, ht := range launch.EncoderHinters {
		_ = t.Encs.TestAddHinter(ht)
	}

	_ = t.Encs.TestAddHinter(AccountValue{})
	_ = t.Encs.TestAddHinter(BaseHal{})
	_ = t.Encs.TestAddHinter(NodeInfo{})
	_ = t.Encs.TestAddHinter(OperationValue{})
	_ = t.Encs.TestAddHinter(Problem{})
	_ = t.Encs.TestAddHinter(currency.AccountHinter)
	_ = t.Encs.TestAddHinter(currency.AddressHinter)
	_ = t.Encs.TestAddHinter(currency.AmountHinter)
	_ = t.Encs.TestAddHinter(currency.CreateAccountsFactHinter)
	_ = t.Encs.TestAddHinter(currency.CreateAccountsItemMultiAmountsHinter)
	_ = t.Encs.TestAddHinter(currency.CreateAccountsItemSingleAmountHinter)
	_ = t.Encs.TestAddHinter(currency.CreateAccountsHinter)
	_ = t.Encs.TestAddHinter(currency.CurrencyDesignHinter)
	_ = t.Encs.TestAddHinter(currency.CurrencyPolicyUpdaterFactHinter)
	_ = t.Encs.TestAddHinter(currency.CurrencyPolicyUpdaterHinter)
	_ = t.Encs.TestAddHinter(currency.CurrencyRegisterFactHinter)
	_ = t.Encs.TestAddHinter(currency.CurrencyRegisterHinter)
	_ = t.Encs.TestAddHinter(currency.FeeOperationFactHinter)
	_ = t.Encs.TestAddHinter(currency.FeeOperationHinter)
	_ = t.Encs.TestAddHinter(currency.FixedFeeerHinter)
	_ = t.Encs.TestAddHinter(currency.GenesisCurrenciesFactHinter)
	_ = t.Encs.TestAddHinter(currency.GenesisCurrenciesHinter)
	_ = t.Encs.TestAddHinter(currency.KeyUpdaterFactHinter)
	_ = t.Encs.TestAddHinter(currency.KeyUpdaterHinter)
	_ = t.Encs.TestAddHinter(currency.AccountKeysHinter)
	_ = t.Encs.TestAddHinter(currency.AccountKeyHinter)
	_ = t.Encs.TestAddHinter(currency.NilFeeerHinter)
	_ = t.Encs.TestAddHinter(currency.RatioFeeerHinter)
	_ = t.Encs.TestAddHinter(currency.TransfersFactHinter)
	_ = t.Encs.TestAddHinter(currency.TransfersItemMultiAmountsHinter)
	_ = t.Encs.TestAddHinter(currency.TransfersItemSingleAmountHinter)
	_ = t.Encs.TestAddHinter(currency.TransfersHinter)
	_ = t.Encs.TestAddHinter(currency.CurrencyPolicyHinter)
	_ = t.Encs.TestAddHinter(currency.SuffrageInflationHinter)

	t.networkID = util.UUID().Bytes()

	t.cid = currency.CurrencyID("SHOWME")
}

func (t *baseTest) MongodbDatabase() *mongodbstorage.Database {
	return t.StorageSupportTest.Database(t.Encs, t.BSONEnc).(isaac.DummyMongodbDatabase).Database
}

func (t *baseTest) Database() (*Database, *mongodbstorage.Database) {
	mst := t.MongodbDatabase()
	st, err := NewDatabase(mst, t.MongodbDatabase())
	t.NoError(err)

	return st, mst
}

func (t *baseTest) newTransfer(sender, receiver base.Address) currency.Transfers {
	token := util.UUID().Bytes()
	items := []currency.TransfersItem{currency.NewTransfersItemSingleAmount(
		receiver,
		currency.MustNewAmount(currency.NewBig(10), t.cid),
	)}
	fact := currency.NewTransfersFact(token, sender, items)

	pk := key.NewBasePrivatekey()
	sig, err := base.NewFactSignature(pk, fact, t.networkID)
	t.NoError(err)

	tf, err := currency.NewTransfers(
		fact,
		[]base.FactSign{base.NewBaseFactSign(pk.Publickey(), sig)},
		util.UUID().String(),
	)
	t.NoError(err)

	return tf
}

func (t *baseTest) newAccount() currency.Account {
	priv := key.NewBasePrivatekey()
	k, err := currency.NewBaseAccountKey(priv.Publickey(), 100)
	t.NoError(err)

	keys, err := currency.NewBaseAccountKeys([]currency.AccountKey{k}, 100)
	t.NoError(err)

	ac, err := currency.NewAccountFromKeys(keys)
	t.NoError(err)

	return ac
}

func (t *baseTest) newAccountState(ac currency.Account, height base.Height) state.State {
	key := currency.StateKeyAccount(ac.Address())
	value, _ := state.NewHintedValue(ac)

	st, err := state.NewStateV0(key, value, height-1)
	t.NoError(err)
	stu := state.NewStateUpdater(st)

	_ = stu.AddOperation(valuehash.RandomSHA256())
	stu = stu.SetHeight(height)
	t.NoError(stu.SetHash(stu.GenerateHash()))
	return stu.GetState()
}

func (t *baseTest) randomBig() currency.Big {
	var i *big.Int
	for {
		bg := big.NewInt(1000)
		n, err := rand.Int(rand.Reader, bg)
		t.NoError(err)

		if n.Cmp(big.NewInt(0)) >= 0 {
			i = n
			break
		}
	}

	return currency.NewBig(i.Int64())
}

func (t *baseTest) newBalanceState(ac currency.Account, height base.Height, am currency.Amount) state.State {
	key := currency.StateKeyBalance(ac.Address(), am.Currency())

	stv0, err := state.NewStateV0(key, nil, height-1)
	t.NoError(err)
	st, err := currency.SetStateBalanceValue(stv0, am)
	t.NoError(err)

	stu := state.NewStateUpdater(st)

	t.NoError(stu.SetHash(stu.GenerateHash()))
	t.NoError(stu.AddOperation(valuehash.RandomSHA256()))
	stu = stu.SetHeight(height)
	t.NoError(stu.SetHash(stu.GenerateHash()))

	return stu.GetState()
}

func (t *baseTest) insertDoc(st *Database, col string, doc mongodbstorage.Doc) interface{} {
	id, err := st.database.Client().Add(col, doc)
	t.NoError(err)

	return id
}

func (t *baseTest) insertAccount(
	st *Database, height base.Height, ac currency.Account, am currency.Amount,
) (AccountValue, []state.State) {
	var va AccountValue
	sts := make([]state.State, 2)
	{
		s := t.newAccountState(ac, height)
		v, err := NewAccountValue(s)
		t.NoError(err)
		va = v

		doc, err := NewAccountDoc(va, t.BSONEnc)
		t.NoError(err)
		t.insertDoc(st, defaultColNameAccount, doc)

		sts[0] = s
	}

	{
		s := t.newBalanceState(ac, height, am)
		doc, err := NewBalanceDoc(s, t.BSONEnc)
		t.NoError(err)
		t.insertDoc(st, defaultColNameBalance, doc)

		sts[1] = s
	}

	va = va.SetBalance([]currency.Amount{am})

	return va, sts
}

func (t *baseTest) compareAccount(a, b interface{}) {
	ua, ok := a.(currency.Account)
	t.True(ok)
	ub, ok := b.(currency.Account)
	t.True(ok)

	t.True(ua.Hash().Equal(ub.Hash()))
	t.True(ua.Address().Equal(ub.Address()))
	t.True(ua.Keys().Equal(ub.Keys()))
}

func (t *baseTest) compareAccountValue(a, b interface{}) {
	ua, ok := a.(AccountValue)
	t.True(ok)
	ub, ok := b.(AccountValue)
	t.True(ok)

	t.compareAccount(ua.Account(), ub.Account())
	t.Equal(ua.Height(), ub.Height())
	t.Equal(ua.PreviousHeight(), ub.PreviousHeight())

	for i := range ua.Balance() {
		t.compareAmount(ua.Balance()[i], ub.Balance()[i])
	}
}

func (t *baseTest) compareOperationValue(a, b interface{}) {
	ua, ok := a.(OperationValue)
	t.True(ok)
	ub, ok := b.(OperationValue)
	t.True(ok)

	uaop := ua.Operation()
	ubop := ub.Operation()

	t.Equal(ua.Height(), ub.Height())
	t.True(localtime.Equal(ua.ConfirmedAt(), ub.ConfirmedAt()))

	t.True(uaop.Hint().Equal(ubop.Hint()))
	t.True(uaop.Hash().Equal(ubop.Hash()))
	t.True(uaop.Fact().Hash().Equal(ubop.Fact().Hash()))
	t.True(localtime.Equal(uaop.LastSignedAt(), ubop.LastSignedAt()))
	t.Equal(ua.InState(), ub.InState())
}

func (t *baseTest) compareAmount(a, b interface{}) {
	ua, ok := a.(currency.Amount)
	t.True(ok)
	ub, ok := b.(currency.Amount)
	t.True(ok)

	t.True(ua.Big().Equal(ub.Big()))
	t.Equal(ua.Currency(), ub.Currency())
}

func (t *baseTest) newBlock(height base.Height, st storage.Database) block.Block {
	var blk block.BlockUpdater
	i, err := block.NewBlockV0(
		block.SuffrageInfoV0{},
		height,
		base.Round(1),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		localtime.UTCNow(),
	)
	t.NoError(err)
	blk = i

	ivp := base.NewVoteproofV0(blk.Height(), blk.Round(), nil, base.ThresholdRatio(100), base.StageINIT)
	avp := base.NewVoteproofV0(blk.Height(), blk.Round(), nil, base.ThresholdRatio(100), base.StageACCEPT)
	blk = blk.SetINITVoteproof(ivp).SetACCEPTVoteproof(avp)

	bd := block.NewBaseBlockdataMap(block.BaseBlockdataMapHint, blk.Height())
	for _, dataType := range block.Blockdata {
		bd, err = bd.SetItem(block.NewBaseBlockdataMapItem(dataType, util.UUID().String(), "file:///"+util.UUID().String()))
		t.NoError(err)
	}
	bd = bd.SetBlock(blk.Hash())
	bd, err = bd.UpdateHash()
	t.NoError(err)

	bs, err := st.NewSession(blk)
	t.NoError(err)
	t.NoError(bs.Commit(context.Background(), bd))

	return blk
}

func (t *baseTest) compareCurrencyDesign(a, b currency.CurrencyDesign) {
	t.compareAmount(a.Amount, b.Amount)
	t.True(a.GenesisAccount().Equal(a.GenesisAccount()))
	t.Equal(a.Policy(), b.Policy())
}
