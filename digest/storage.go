package digest

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
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
	"golang.org/x/xerrors"
)

var maxLimit int64 = 50

var (
	defaultColNameAccount   = "digest_ac"
	defaultColNameBalance   = "digest_bl"
	defaultColNameOperation = "digest_op"
)

var DigestStorageLastBlockKey = "digest_last_block"

type Storage struct {
	sync.RWMutex
	*logging.Logging
	mitum     *mongodbstorage.Storage
	storage   *mongodbstorage.Storage
	readonly  bool
	lastBlock base.Height
}

func NewStorage(mitum *mongodbstorage.Storage, st *mongodbstorage.Storage) (*Storage, error) {
	nst := &Storage{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "digest-mongodb-storage")
		}),
		mitum:     mitum,
		storage:   st,
		lastBlock: base.NilHeight,
	}
	_ = nst.SetLogger(mitum.Log())

	return nst, nil
}

func NewReadonlyStorage(mitum *mongodbstorage.Storage, st *mongodbstorage.Storage) (*Storage, error) {
	if st, err := NewStorage(mitum, st); err != nil {
		return nil, err
	} else {
		st.readonly = true

		return st, nil
	}
}

func (st *Storage) New() (*Storage, error) {
	if st.readonly {
		return nil, xerrors.Errorf("readonly mode")
	}

	if nst, err := st.storage.New(); err != nil {
		return nil, err
	} else {
		return NewStorage(st.mitum, nst)
	}
}

func (st *Storage) Readonly() bool {
	return st.readonly
}

func (st *Storage) Close() error {
	return st.storage.Close()
}

func (st *Storage) Initialize() error {
	st.Lock()
	defer st.Unlock()

	switch h, found, err := loadLastBlock(st); {
	case err != nil:
		return xerrors.Errorf("failed to get last block for digest: %w", err)
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

func (st *Storage) createIndex() error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	for col, models := range defaultIndexes {
		if err := st.storage.CreateIndex(col, models, indexPrefix); err != nil {
			return err
		}
	}

	return nil
}

func (st *Storage) LastBlock() base.Height {
	st.RLock()
	defer st.RUnlock()

	return st.lastBlock
}

func (st *Storage) SetLastBlock(height base.Height) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	st.Lock()
	defer st.Unlock()

	if height <= st.lastBlock {
		return nil
	}

	return st.setLastBlock(height)
}

func (st *Storage) setLastBlock(height base.Height) error {
	if err := st.storage.SetInfo(DigestStorageLastBlockKey, height.Bytes()); err != nil {
		st.Log().Debug().Hinted("height", height).Msg("failed to set last block")

		return err
	} else {
		st.lastBlock = height
		st.Log().Debug().Hinted("height", height).Msg("set last block")

		return nil
	}
}

func (st *Storage) Clean() error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	st.Lock()
	defer st.Unlock()

	return st.clean()
}

func (st *Storage) clean() error {
	for _, col := range []string{
		defaultColNameAccount,
		defaultColNameBalance,
		defaultColNameOperation,
	} {
		if err := st.storage.Client().Collection(col).Drop(context.Background()); err != nil {
			return storage.WrapStorageError(err)
		}

		st.Log().Debug().Str("collection", col).Msg("drop collection by height")
	}

	if err := st.setLastBlock(base.NilHeight); err != nil {
		return err
	}

	st.Log().Debug().Msg("clean digest")

	return nil
}

func (st *Storage) CleanByHeight(height base.Height) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	st.Lock()
	defer st.Unlock()

	return st.cleanByHeight(height)
}

func (st *Storage) cleanByHeight(height base.Height) error {
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
		res, err := st.storage.Client().Collection(col).BulkWrite(
			context.Background(),
			[]mongo.WriteModel{removeByHeight},
			opts,
		)
		if err != nil {
			return storage.WrapStorageError(err)
		}

		st.Log().Debug().Str("collection", col).Interface("result", res).Msg("clean collection by height")
	}

	return st.setLastBlock(height - 1)
}

func (st *Storage) ManifestByHeight(height base.Height) (block.Manifest, bool, error) {
	return st.mitum.ManifestByHeight(height)
}

func (st *Storage) Manifest(h valuehash.Hash) (block.Manifest, bool, error) {
	return st.mitum.Manifest(h)
}

// Manifests returns block.Manifests by it's order, height.
func (st *Storage) Manifests(
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
func (st *Storage) OperationsByAddress(
	address base.Address,
	load,
	reverse bool,
	offset string,
	limit int64,
	callback func(valuehash.Hash /* fact hash */, OperationValue) (bool, error),
) error {
	var filter bson.M
	if f, err := buildOperationsFilterByAddress(address, offset, reverse); err != nil {
		return err
	} else {
		filter = f
	}

	var sr int = 1
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

	return st.storage.Client().Find(
		context.Background(),
		defaultColNameOperation,
		filter,
		func(cursor *mongo.Cursor) (bool, error) {
			if !load {
				if h, err := loadOperationHash(cursor.Decode); err != nil {
					return false, err
				} else {
					return callback(h, OperationValue{})
				}
			}

			if va, err := loadOperation(cursor.Decode, st.storage.Encoders()); err != nil {
				return false, err
			} else {
				return callback(va.Operation().Fact().Hash(), va)
			}
		},
		opt,
	)
}

// Operation returns operation.Operation. If load is false, just returns nil
// Operation.
func (st *Storage) Operation(
	h valuehash.Hash, /* fact hash */
	load bool,
) (OperationValue, bool /* exists */, error) {
	if !load {
		exists, err := st.storage.Client().Exists(defaultColNameOperation, util.NewBSONFilter("fact", h).D())
		return OperationValue{}, exists, err
	}

	var va OperationValue
	if err := st.storage.Client().GetByFilter(
		defaultColNameOperation,
		util.NewBSONFilter("fact", h).D(),
		func(res *mongo.SingleResult) error {
			if !load {
				return nil
			}

			if i, err := loadOperation(res.Decode, st.storage.Encoders()); err != nil {
				return err
			} else {
				va = i

				return nil
			}
		},
	); err != nil {
		if xerrors.Is(err, storage.NotFoundError) {
			return OperationValue{}, false, nil
		}

		return OperationValue{}, false, err
	} else {
		return va, true, nil
	}
}

// Operations returns operation.Operations by it's order, height and index.
func (st *Storage) Operations(
	filter bson.M,
	load bool,
	reverse bool,
	limit int64,
	callback func(valuehash.Hash /* fact hash */, OperationValue) (bool, error),
) error {
	var sr int = 1
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

	return st.storage.Client().Find(
		context.Background(),
		defaultColNameOperation,
		filter,
		func(cursor *mongo.Cursor) (bool, error) {
			if !load {
				if h, err := loadOperationHash(cursor.Decode); err != nil {
					return false, err
				} else {
					return callback(h, OperationValue{})
				}
			}

			if va, err := loadOperation(cursor.Decode, st.storage.Encoders()); err != nil {
				return false, err
			} else {
				return callback(va.Operation().Fact().Hash(), va)
			}
		},
		opt,
	)
}

// Account returns AccountValue.
func (st *Storage) Account(a base.Address) (AccountValue, bool /* exists */, error) {
	var rs AccountValue
	if err := st.storage.Client().GetByFilter(
		defaultColNameAccount,
		util.NewBSONFilter("address", currency.StateAddressKeyPrefix(a)).D(),
		func(res *mongo.SingleResult) error {
			if i, err := loadAccountValue(res.Decode, st.storage.Encoders()); err != nil {
				return err
			} else {
				rs = i

				return nil
			}
		},
		options.FindOne().SetSort(util.NewBSONFilter("height", -1).D()),
	); err != nil {
		if xerrors.Is(err, storage.NotFoundError) {
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

func (st *Storage) balance(a base.Address) ([]currency.Amount, base.Height, base.Height, error) {
	var lastHeight, previousHeight base.Height = base.NilHeight, base.NilHeight
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
		if err := st.storage.Client().GetByFilter(
			defaultColNameBalance,
			q,
			func(res *mongo.SingleResult) error {
				if i, err := loadBalance(res.Decode, st.storage.Encoders()); err != nil {
					return err
				} else {
					sta = i

					return nil
				}
			},
			options.FindOne().SetSort(util.NewBSONFilter("height", -1).D()),
		); err != nil {
			if xerrors.Is(err, storage.NotFoundError) {
				break
			}

			return nil, lastHeight, previousHeight, err
		}

		if i, err := currency.StateBalanceValue(sta); err != nil {
			return nil, lastHeight, previousHeight, err
		} else {
			amm[i.Currency()] = i

			cids = append(cids, i.Currency().String())
		}

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

func loadLastBlock(st *Storage) (base.Height, bool, error) {
	switch b, found, err := st.storage.Info(DigestStorageLastBlockKey); {
	case err != nil:
		return base.NilHeight, false, xerrors.Errorf("failed to get last block for digest: %w", err)
	case !found:
		return base.NilHeight, false, nil
	default:
		if h, err := base.NewHeightFromBytes(b); err != nil {
			return base.NilHeight, false, err
		} else {
			return h, true, nil
		}
	}
}

func parseOffset(s string) (base.Height, uint64, error) {
	if n := strings.SplitN(s, ",", 2); n == nil {
		return base.NilHeight, 0, xerrors.Errorf("invalid offset string: %q", s)
	} else if len(n) < 2 {
		return base.NilHeight, 0, xerrors.Errorf("invalid offset, %q", s)
	} else if h, err := base.NewHeightFromString(n[0]); err != nil {
		return base.NilHeight, 0, xerrors.Errorf("invalid height of offset: %w", err)
	} else if u, err := strconv.ParseUint(n[1], 10, 64); err != nil {
		return base.NilHeight, 0, xerrors.Errorf("invalid index of offset: %w", err)
	} else {
		return h, u, nil
	}
}

func buildOffset(height base.Height, index uint64) string {
	return fmt.Sprintf("%d,%d", height, index)
}

func buildOperationsFilterByAddress(address base.Address, offset string, reverse bool) (bson.M, error) {
	filter := bson.M{"addresses": bson.M{"$in": []string{currency.StateAddressKeyPrefix(address)}}}
	if len(offset) > 0 {
		var height base.Height
		var index uint64
		if h, i, err := parseOffset(offset); err != nil {
			return nil, err
		} else {
			height = h
			index = i
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
