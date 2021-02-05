// +build test mongodb

package digest

import (
	"context"
	"crypto/rand"
	"math/big"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/storage"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum-currency/currency"
)

//lint:ignore U1000 debugging for test
var log logging.Logger

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	l := zerolog.
		New(os.Stderr).
		With().
		Timestamp().
		Caller().
		Stack().
		Logger().Level(zerolog.DebugLevel)

	log = logging.NewLogger(&l, true)
}

type baseTest struct { // nolint: unused
	suite.Suite
	isaac.StorageSupportTest
	networkID base.NetworkID
	cid       currency.CurrencyID
}

func (t *baseTest) SetupSuite() {
	t.DBType = "mongodb"
	t.StorageSupportTest.SetupSuite()

	for _, ht := range process.DefaultHinters {
		_ = t.Encs.AddHinter(ht)
	}

	_ = t.Encs.AddHinter(AccountValue{})
	_ = t.Encs.AddHinter(BaseHal{})
	_ = t.Encs.AddHinter(NodeInfo{})
	_ = t.Encs.AddHinter(OperationValue{})
	_ = t.Encs.AddHinter(Problem{})
	_ = t.Encs.AddHinter(currency.Account{})
	_ = t.Encs.AddHinter(currency.Address(""))
	_ = t.Encs.AddHinter(currency.Amount{})
	_ = t.Encs.AddHinter(currency.CreateAccountsFact{})
	_ = t.Encs.AddHinter(currency.CreateAccountsItemMultiAmountsHinter)
	_ = t.Encs.AddHinter(currency.CreateAccountsItemSingleAmountHinter)
	_ = t.Encs.AddHinter(currency.CreateAccounts{})
	_ = t.Encs.AddHinter(currency.CurrencyDesign{})
	_ = t.Encs.AddHinter(currency.CurrencyPolicyUpdaterFact{})
	_ = t.Encs.AddHinter(currency.CurrencyPolicyUpdater{})
	_ = t.Encs.AddHinter(currency.CurrencyRegisterFact{})
	_ = t.Encs.AddHinter(currency.CurrencyRegister{})
	_ = t.Encs.AddHinter(currency.FeeOperationFact{})
	_ = t.Encs.AddHinter(currency.FeeOperation{})
	_ = t.Encs.AddHinter(currency.FixedFeeer{})
	_ = t.Encs.AddHinter(currency.GenesisCurrenciesFact{})
	_ = t.Encs.AddHinter(currency.GenesisCurrencies{})
	_ = t.Encs.AddHinter(currency.KeyUpdaterFact{})
	_ = t.Encs.AddHinter(currency.KeyUpdater{})
	_ = t.Encs.AddHinter(currency.Keys{})
	_ = t.Encs.AddHinter(currency.Key{})
	_ = t.Encs.AddHinter(currency.NilFeeer{})
	_ = t.Encs.AddHinter(currency.RatioFeeer{})
	_ = t.Encs.AddHinter(currency.TransfersFact{})
	_ = t.Encs.AddHinter(currency.TransfersItemMultiAmountsHinter)
	_ = t.Encs.AddHinter(currency.TransfersItemSingleAmountHinter)
	_ = t.Encs.AddHinter(currency.Transfers{})
	_ = t.Encs.AddHinter(currency.CurrencyPolicy{})
	_ = t.Encs.AddHinter(key.BTCPublickeyHinter)
	_ = t.Encs.AddHinter(operation.BaseFactSign{})

	t.networkID = util.UUID().Bytes()

	t.cid = currency.CurrencyID("SHOWME")
}

func (t *baseTest) MongodbStorage() *mongodbstorage.Storage {
	return t.StorageSupportTest.Storage(t.Encs, t.BSONEnc).(isaac.DummyMongodbStorage).Storage
}

func (t *baseTest) Storage() (*Storage, *mongodbstorage.Storage) {
	mst := t.MongodbStorage()
	st, err := NewStorage(mst, t.MongodbStorage())
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

	pk := key.MustNewEtherPrivatekey()
	sig, err := operation.NewFactSignature(pk, fact, t.networkID)
	t.NoError(err)

	tf, err := currency.NewTransfers(
		fact,
		[]operation.FactSign{operation.NewBaseFactSign(pk.Publickey(), sig)},
		util.UUID().String(),
	)
	t.NoError(err)

	return tf
}

func (t *baseTest) newAccount() currency.Account {
	priv := key.MustNewBTCPrivatekey()
	k, err := currency.NewKey(priv.Publickey(), 100)
	t.NoError(err)

	keys, err := currency.NewKeys([]currency.Key{k}, 100)
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

func (t *baseTest) insertDoc(st *Storage, col string, doc mongodbstorage.Doc) interface{} {
	id, err := st.storage.Client().Add(col, doc)
	t.NoError(err)

	return id
}

func (t *baseTest) insertAccount(
	st *Storage, height base.Height, ac currency.Account, am currency.Amount,
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

func (t *baseTest) newBlock(height base.Height, st storage.Storage) block.Block {
	blk, err := block.NewBlockV0(
		block.SuffrageInfoV0{},
		height,
		base.Round(1),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		localtime.Now(),
	)
	t.NoError(err)

	bs, err := st.OpenBlockStorage(blk)
	t.NoError(err)
	t.NoError(bs.Commit(context.Background()))

	return blk
}

func (t *baseTest) compareCurrencyDesign(a, b currency.CurrencyDesign) {
	t.compareAmount(a.Amount, b.Amount)
	t.True(a.GenesisAccount().Equal(a.GenesisAccount()))
	t.Equal(a.Policy(), b.Policy())
}
