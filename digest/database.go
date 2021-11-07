package digest

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var maxLimit int64 = 50

var (
	defaultColNameAccount   = "digest_ac"
	defaultColNameBalance   = "digest_bl"
	defaultColNameOperation = "digest_op"
)

var DigestStorageLastBlockKey = "digest_last_block"

type Database struct {
	sync.RWMutex
	*logging.Logging
	mitum     *mongodbstorage.Database
	database  *mongodbstorage.Database
	readonly  bool
	lastBlock base.Height
}

func NewDatabase(mitum *mongodbstorage.Database, st *mongodbstorage.Database) (*Database, error) {
	nst := &Database{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "digest-mongodb-database")
		}),
		mitum:     mitum,
		database:  st,
		lastBlock: base.NilHeight,
	}
	_ = nst.SetLogging(mitum.Logging)

	return nst, nil
}

func NewReadonlyDatabase(mitum *mongodbstorage.Database, st *mongodbstorage.Database) (*Database, error) {
	nst, err := NewDatabase(mitum, st)
	if err != nil {
		return nil, err
	}
	nst.readonly = true

	return nst, nil
}

func (st *Database) New() (*Database, error) {
	if st.readonly {
		return nil, errors.Errorf("readonly mode")
	}

	nst, err := st.database.New()
	if err != nil {
		return nil, err
	}
	return NewDatabase(st.mitum, nst)
}

func (st *Database) Readonly() bool {
	return st.readonly
}

func (st *Database) Close() error {
	return st.database.Close()
}

func (st *Database) Initialize() error {
	st.Lock()
	defer st.Unlock()

	switch h, found, err := loadLastBlock(st); {
	case err != nil:
		return errors.Wrap(err, "failed to get last block for digest")
	case !found:
		st.lastBlock = base.NilHeight
		st.Log().Debug().Msg("last block for digest not found")
	default:
		st.lastBlock = h

		if !st.readonly {
			if err := st.createIndex(); err != nil {
				return err
			}

			if err := st.cleanByHeight(h + 1); err != nil {
				return err
			}
		}
	}

	return nil
}

func (st *Database) createIndex() error {
	if st.readonly {
		return errors.Errorf("readonly mode")
	}

	for col, models := range defaultIndexes {
		if err := st.database.CreateIndex(col, models, indexPrefix); err != nil {
			return err
		}
	}

	return nil
}

func (st *Database) LastBlock() base.Height {
	st.RLock()
	defer st.RUnlock()

	return st.lastBlock
}

func (st *Database) SetLastBlock(height base.Height) error {
	if st.readonly {
		return errors.Errorf("readonly mode")
	}

	st.Lock()
	defer st.Unlock()

	if height <= st.lastBlock {
		return nil
	}

	return st.setLastBlock(height)
}

func (st *Database) setLastBlock(height base.Height) error {
	if err := st.database.SetInfo(DigestStorageLastBlockKey, height.Bytes()); err != nil {
		st.Log().Debug().Int64("height", height.Int64()).Msg("failed to set last block")

		return err
	}
	st.lastBlock = height
	st.Log().Debug().Int64("height", height.Int64()).Msg("set last block")

	return nil
}

func (st *Database) Clean() error {
	if st.readonly {
		return errors.Errorf("readonly mode")
	}

	st.Lock()
	defer st.Unlock()

	return st.clean()
}

func (st *Database) clean() error {
	for _, col := range []string{
		defaultColNameAccount,
		defaultColNameBalance,
		defaultColNameOperation,
	} {
		if err := st.database.Client().Collection(col).Drop(context.Background()); err != nil {
			return storage.MergeStorageError(err)
		}

		st.Log().Debug().Str("collection", col).Msg("drop collection by height")
	}

	if err := st.setLastBlock(base.NilHeight); err != nil {
		return err
	}

	st.Log().Debug().Msg("clean digest")

	return nil
}

func (st *Database) CleanByHeight(height base.Height) error {
	if st.readonly {
		return errors.Errorf("readonly mode")
	}

	st.Lock()
	defer st.Unlock()

	return st.cleanByHeight(height)
}

func (st *Database) cleanByHeight(height base.Height) error {
	if height <= base.PreGenesisHeight+1 {
		return st.clean()
	}

	opts := options.BulkWrite().SetOrdered(true)
	removeByHeight := mongo.NewDeleteManyModel().SetFilter(bson.M{"height": bson.M{"$gte": height}})

	for _, col := range []string{
		defaultColNameAccount,
		defaultColNameBalance,
		defaultColNameOperation,
	} {
		res, err := st.database.Client().Collection(col).BulkWrite(
			context.Background(),
			[]mongo.WriteModel{removeByHeight},
			opts,
		)
		if err != nil {
			return storage.MergeStorageError(err)
		}

		st.Log().Debug().Str("collection", col).Interface("result", res).Msg("clean collection by height")
	}

	return st.setLastBlock(height - 1)
}

func (st *Database) ManifestByHeight(height base.Height) (block.Manifest, bool, error) {
	return st.mitum.ManifestByHeight(height)
}

func (st *Database) Manifest(h valuehash.Hash) (block.Manifest, bool, error) {
	return st.mitum.Manifest(h)
}

// Manifests returns block.Manifests by it's order, height.
func (st *Database) Manifests(
	load bool,
	reverse bool,
	offset base.Height,
	limit int64,
	callback func(base.Height, valuehash.Hash /* block hash */, block.Manifest) (bool, error),
) error {
	var filter bson.M
	if offset > base.NilHeight {
		if reverse {
			filter = bson.M{"height": bson.M{"$lt": offset}}
		} else {
			filter = bson.M{"height": bson.M{"$gt": offset}}
		}
	}

	return st.mitum.Manifests(
		filter,
		load,
		reverse,
		limit,
		callback,
	)
}

// OperationsByAddress finds the operation.Operations, which are related with
// the given Address. The returned valuehash.Hash is the
// operation.Operation.Fact().Hash().
// *    load:if true, load operation.Operation and returns it. If not, just hash will be returned
// * reverse: order by height; if true, higher height will be returned first.
// *  offset: returns from next of offset, usually it is combination of
// "<height>,<fact>".
func (st *Database) OperationsByAddress(
	address base.Address,
	load,
	reverse bool,
	offset string,
	limit int64,
	callback func(valuehash.Hash /* fact hash */, OperationValue) (bool, error),
) error {
	filter, err := buildOperationsFilterByAddress(address, offset, reverse)
	if err != nil {
		return err
	}

	sr := 1
	if reverse {
		sr = -1
	}

	opt := options.Find().SetSort(
		util.NewBSONFilter("height", sr).Add("index", sr).D(),
	)

	switch {
	case limit <= 0: // no limit
	case limit > maxLimit:
		opt = opt.SetLimit(maxLimit)
	default:
		opt = opt.SetLimit(limit)
	}

	if !load {
		opt = opt.SetProjection(bson.M{"fact": 1})
	}

	return st.database.Client().Find(
		context.Background(),
		defaultColNameOperation,
		filter,
		func(cursor *mongo.Cursor) (bool, error) {
			if !load {
				h, err := LoadOperationHash(cursor.Decode)
				if err != nil {
					return false, err
				}
				return callback(h, OperationValue{})
			}

			va, err := LoadOperation(cursor.Decode, st.database.Encoders())
			if err != nil {
				return false, err
			}
			return callback(va.Operation().Fact().Hash(), va)
		},
		opt,
	)
}

// Operation returns operation.Operation. If load is false, just returns nil
// Operation.
func (st *Database) Operation(
	h valuehash.Hash, /* fact hash */
	load bool,
) (OperationValue, bool /* exists */, error) {
	if !load {
		exists, err := st.database.Client().Exists(defaultColNameOperation, util.NewBSONFilter("fact", h).D())
		return OperationValue{}, exists, err
	}

	var va OperationValue
	if err := st.database.Client().GetByFilter(
		defaultColNameOperation,
		util.NewBSONFilter("fact", h).D(),
		func(res *mongo.SingleResult) error {
			if !load {
				return nil
			}

			i, err := LoadOperation(res.Decode, st.database.Encoders())
			if err != nil {
				return err
			}
			va = i

			return nil
		},
	); err != nil {
		if errors.Is(err, util.NotFoundError) {
			return OperationValue{}, false, nil
		}

		return OperationValue{}, false, err
	}
	return va, true, nil
}

// Operations returns operation.Operations by it's order, height and index.
func (st *Database) Operations(
	filter bson.M,
	load bool,
	reverse bool,
	limit int64,
	callback func(valuehash.Hash /* fact hash */, OperationValue) (bool, error),
) error {
	sr := 1
	if reverse {
		sr = -1
	}

	opt := options.Find().SetSort(
		util.NewBSONFilter("height", sr).Add("index", sr).D(),
	)

	switch {
	case limit <= 0: // no limit
	case limit > maxLimit:
		opt = opt.SetLimit(maxLimit)
	default:
		opt = opt.SetLimit(limit)
	}

	if !load {
		opt = opt.SetProjection(bson.M{"fact": 1})
	}

	return st.database.Client().Find(
		context.Background(),
		defaultColNameOperation,
		filter,
		func(cursor *mongo.Cursor) (bool, error) {
			if !load {
				h, err := LoadOperationHash(cursor.Decode)
				if err != nil {
					return false, err
				}
				return callback(h, OperationValue{})
			}

			va, err := LoadOperation(cursor.Decode, st.database.Encoders())
			if err != nil {
				return false, err
			}
			return callback(va.Operation().Fact().Hash(), va)
		},
		opt,
	)
}

// Account returns AccountValue.
func (st *Database) Account(a base.Address) (AccountValue, bool /* exists */, error) {
	var rs AccountValue
	if err := st.database.Client().GetByFilter(
		defaultColNameAccount,
		util.NewBSONFilter("address", currency.StateAddressKeyPrefix(a)).D(),
		func(res *mongo.SingleResult) error {
			i, err := LoadAccountValue(res.Decode, st.database.Encoders())
			if err != nil {
				return err
			}
			rs = i

			return nil
		},
		options.FindOne().SetSort(util.NewBSONFilter("height", -1).D()),
	); err != nil {
		if errors.Is(err, util.NotFoundError) {
			return rs, false, nil
		}

		return rs, false, err
	}

	// NOTE load balance
	switch am, lastHeight, previousHeight, err := st.balance(a); {
	case err != nil:
		return rs, false, err
	default:
		rs = rs.SetBalance(am).
			SetHeight(lastHeight).
			SetPreviousHeight(previousHeight)
	}

	return rs, true, nil
}

// AccountsByPublickey finds Accounts, which are related with the given
// Publickey.
// *  offset: returns from next of offset, usually it is "<height>,<address>".
func (st *Database) AccountsByPublickey(
	pub key.Publickey,
	loadBalance bool,
	offsetHeight base.Height,
	offsetAddress string,
	limit int64,
	callback func(AccountValue) (bool, error),
) error {
	if offsetHeight <= base.NilHeight {
		return errors.Errorf("offset height should be over nil height")
	}

	filter := buildAccountsFilterByPublickey(pub)
	filter["height"] = bson.M{"$lte": offsetHeight}

	var sas []string
	switch i, err := st.addressesByPublickey(filter); {
	case err != nil:
		return err
	default:
		sas = i
	}

	if len(sas) < 1 {
		return nil
	}

	var filteredAddress []string
	if len(offsetAddress) < 1 {
		filteredAddress = sas
	} else {
		var found bool
		for i := range sas {
			a := sas[i]
			if !found {
				if offsetAddress == a {
					found = true
				}

				continue
			}

			filteredAddress = append(filteredAddress, a)
		}
	}

	if len(filteredAddress) < 1 {
		return nil
	}

end:
	for i := int64(0); i < int64(math.Ceil(float64(len(filteredAddress))/50.0)); i++ {
		l := (i + 1) + 50
		if n := int64(len(filteredAddress)); l > n {
			l = n
		}

		limited := filteredAddress[i*50 : l]
		switch done, err := st.filterAccountByPublickey(
			pub, limited, limit, loadBalance, callback,
		); {
		case err != nil:
			return err
		case done:
			break end
		}
	}

	return nil
}

func (st *Database) balance(a base.Address) ([]currency.Amount, base.Height, base.Height, error) {
	lastHeight, previousHeight := base.NilHeight, base.NilHeight
	var cids []string

	amm := map[currency.CurrencyID]currency.Amount{}
	for {
		filter := util.NewBSONFilter("address", currency.StateAddressKeyPrefix(a))

		var q primitive.D
		if len(cids) < 1 {
			q = filter.D()
		} else {
			q = filter.Add("currency", bson.M{"$nin": cids}).D()
		}

		var sta state.State
		if err := st.database.Client().GetByFilter(
			defaultColNameBalance,
			q,
			func(res *mongo.SingleResult) error {
				i, err := LoadBalance(res.Decode, st.database.Encoders())
				if err != nil {
					return err
				}
				sta = i

				return nil
			},
			options.FindOne().SetSort(util.NewBSONFilter("height", -1).D()),
		); err != nil {
			if errors.Is(err, util.NotFoundError) {
				break
			}

			return nil, lastHeight, previousHeight, err
		}

		i, err := currency.StateBalanceValue(sta)
		if err != nil {
			return nil, lastHeight, previousHeight, err
		}
		amm[i.Currency()] = i

		cids = append(cids, i.Currency().String())

		if h := sta.Height(); h > lastHeight {
			lastHeight = h
			previousHeight = sta.PreviousHeight()
		}
	}

	ams := make([]currency.Amount, len(amm))
	var i int
	for k := range amm {
		ams[i] = amm[k]
		i++
	}

	return ams, lastHeight, previousHeight, nil
}

func (st *Database) topHeightByPublickey(pub key.Publickey) (base.Height, error) {
	var sas []string
	switch r, err := st.database.Client().Collection(defaultColNameAccount).Distinct(
		context.Background(),
		"address",
		buildAccountsFilterByPublickey(pub),
	); {
	case err != nil:
		return base.NilHeight, err
	case len(r) < 1:
		return base.NilHeight, err
	default:
		sas = make([]string, len(r))
		for i := range r {
			sas[i] = r[i].(string)
		}
	}

	var top base.Height
	for i := int64(0); i < int64(math.Ceil(float64(len(sas))/50.0)); i++ {
		l := (i + 1) + 50
		if n := int64(len(sas)); l > n {
			l = n
		}

		switch h, err := st.partialTopHeightByPublickey(sas[i*50 : l]); {
		case err != nil:
			return base.NilHeight, err
		case top <= base.NilHeight:
			top = h
		case h > top:
			top = h
		}
	}

	return top, nil
}

func (st *Database) partialTopHeightByPublickey(as []string) (base.Height, error) {
	var top base.Height
	err := st.database.Client().Find(
		context.Background(),
		defaultColNameAccount,
		bson.M{"address": bson.M{"$in": as}},
		func(cursor *mongo.Cursor) (bool, error) {
			h, err := loadHeightDoc(cursor.Decode)
			if err != nil {
				return false, err
			}

			top = h

			return false, nil
		},
		options.Find().
			SetSort(util.NewBSONFilter("height", -1).D()).
			SetLimit(1),
	)

	return top, err
}

func (st *Database) addressesByPublickey(filter bson.M) ([]string, error) {
	r, err := st.database.Client().Collection(defaultColNameAccount).Distinct(context.Background(), "address", filter)
	if err != nil {
		return nil, storage.MergeStorageError(errors.Wrap(err, "failed to get distinct addresses"))
	}

	if len(r) < 1 {
		return nil, nil
	}

	sas := make([]string, len(r))
	for i := range r {
		sas[i] = r[i].(string)
	}

	sort.Strings(sas)

	return sas, nil
}

func (st *Database) filterAccountByPublickey(
	pub key.Publickey,
	addresses []string,
	limit int64,
	loadBalance bool,
	callback func(AccountValue) (bool, error),
) (bool, error) {
	filter := bson.M{"address": bson.M{"$in": addresses}}

	var lastAddress string
	var called int64
	var stopped bool
	if err := st.database.Client().Find(
		context.Background(),
		defaultColNameAccount,
		filter,
		func(cursor *mongo.Cursor) (bool, error) {
			if called == limit {
				return false, nil
			}

			doc, err := loadBriefAccountDoc(cursor.Decode)
			if err != nil {
				return false, err
			}

			if len(lastAddress) > 0 {
				if lastAddress == doc.Address {
					return true, nil
				}
			}
			lastAddress = doc.Address

			if !doc.pubExists(pub) {
				return true, nil
			}

			va, err := LoadAccountValue(cursor.Decode, st.database.Encoders())
			if err != nil {
				return false, err
			}

			if loadBalance { // NOTE load balance
				switch am, lastHeight, previousHeight, err := st.balance(va.Account().Address()); {
				case err != nil:
					return false, err
				default:
					va = va.SetBalance(am).
						SetHeight(lastHeight).
						SetPreviousHeight(previousHeight)
				}
			}

			called++
			switch keep, err := callback(va); {
			case err != nil:
				return false, err
			case !keep:
				stopped = true

				return false, nil
			default:
				return true, nil
			}
		},
		options.Find().SetSort(util.NewBSONFilter("address", 1).Add("height", -1).D()),
	); err != nil {
		return false, err
	}

	return stopped || called == limit, nil
}

func loadLastBlock(st *Database) (base.Height, bool, error) {
	switch b, found, err := st.database.Info(DigestStorageLastBlockKey); {
	case err != nil:
		return base.NilHeight, false, errors.Wrap(err, "failed to get last block for digest")
	case !found:
		return base.NilHeight, false, nil
	default:
		h, err := base.NewHeightFromBytes(b)
		if err != nil {
			return base.NilHeight, false, err
		}
		return h, true, nil
	}
}

func parseOffset(s string) (base.Height, uint64, error) {
	if n := strings.SplitN(s, ",", 2); n == nil {
		return base.NilHeight, 0, errors.Errorf("invalid offset string: %q", s)
	} else if len(n) < 2 {
		return base.NilHeight, 0, errors.Errorf("invalid offset, %q", s)
	} else if h, err := base.NewHeightFromString(n[0]); err != nil {
		return base.NilHeight, 0, errors.Wrap(err, "invalid height of offset")
	} else if u, err := strconv.ParseUint(n[1], 10, 64); err != nil {
		return base.NilHeight, 0, errors.Wrap(err, "invalid index of offset")
	} else {
		return h, u, nil
	}
}

func buildOffset(height base.Height, index uint64) string {
	return fmt.Sprintf("%d,%d", height, index)
}

func buildOperationsFilterByAddress(address base.Address, offset string, reverse bool) (bson.M, error) {
	filter := bson.M{"addresses": bson.M{"$in": []string{currency.RawTypeString(address)}}}
	if len(offset) > 0 {
		height, index, err := parseOffset(offset)
		if err != nil {
			return nil, err
		}

		if reverse {
			filter["$or"] = []bson.M{
				{"height": bson.M{"$lt": height}},
				{"$and": []bson.M{
					{"height": height},
					{"index": bson.M{"$lt": index}},
				}},
			}
		} else {
			filter["$or"] = []bson.M{
				{"height": bson.M{"$gt": height}},
				{"$and": []bson.M{
					{"height": height},
					{"index": bson.M{"$gt": index}},
				}},
			}
		}
	}

	return filter, nil
}

func parseOffsetByString(s string) (base.Height, string, error) {
	var a, b string
	switch n := strings.SplitN(s, ",", 2); {
	case n == nil:
		return base.NilHeight, "", errors.Errorf("invalid offset string: %q", s)
	case len(n) < 2:
		return base.NilHeight, "", errors.Errorf("invalid offset, %q", s)
	default:
		a = n[0]
		b = n[1]
	}

	h, err := base.NewHeightFromString(a)
	if err != nil {
		return base.NilHeight, "", errors.Wrap(err, "invalid height of offset")
	}

	return h, b, nil
}

func buildOffsetByString(height base.Height, s string) string {
	return fmt.Sprintf("%d,%s", height, s)
}

func buildAccountsFilterByPublickey(pub key.Publickey) bson.M {
	return bson.M{"pubs": bson.M{"$in": []string{currency.RawTypeString(pub)}}}
}

type heightDoc struct {
	H base.Height `bson:"height"`
}

func loadHeightDoc(decoder func(interface{}) error) (base.Height, error) {
	var h heightDoc
	if err := decoder(&h); err != nil {
		return base.NilHeight, err
	}

	return h.H, nil
}

type briefAccountDoc struct {
	ID      primitive.ObjectID `bson:"_id"`
	Address string             `bson:"address"`
	Pubs    []string           `bson:"pubs"`
	Height  base.Height        `bson:"height"`
}

func (doc briefAccountDoc) pubExists(k key.Key) bool {
	if len(doc.Pubs) < 1 {
		return false
	}

	for i := range doc.Pubs {
		if currency.RawTypeString(k) == doc.Pubs[i] {
			return true
		}
	}

	return false
}

func loadBriefAccountDoc(decoder func(interface{}) error) (briefAccountDoc, error) {
	var a briefAccountDoc
	if err := decoder(&a); err != nil {
		return a, err
	}

	return a, nil
}
