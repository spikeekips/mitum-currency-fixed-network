package digest

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/valuehash"
)

var bulkWriteLimit = 500

type BlockStorage struct {
	sync.RWMutex
	block           block.Block
	st              *Storage
	inStates        map[string]struct{}
	operationModels []mongo.WriteModel
	accountModels   []mongo.WriteModel
	balanceModels   []mongo.WriteModel
	statesValue     *sync.Map
}

func NewBlockStorage(st *Storage, blk block.Block) (*BlockStorage, error) {
	var nst *Storage
	if n, err := st.New(); err != nil {
		return nil, err
	} else {
		nst = n
	}

	return &BlockStorage{
		st:          nst,
		block:       blk,
		statesValue: &sync.Map{},
	}, nil
}

func (bs *BlockStorage) Prepare() error {
	bs.Lock()
	defer bs.Unlock()

	if err := bs.prepareOperationsTree(); err != nil {
		return err
	}

	if err := bs.prepareOperations(); err != nil {
		return err
	}

	if err := bs.prepareAccounts(); err != nil {
		return err
	}

	return nil
}

func (bs *BlockStorage) Commit(ctx context.Context) error {
	bs.Lock()
	defer bs.Unlock()

	started := time.Now()
	defer func() {
		bs.statesValue.Store("commit", time.Since(started))

		_ = bs.close()
	}()

	if err := bs.st.CleanByHeight(bs.block.Height()); err != nil {
		return err
	}

	if err := bs.writeModels(ctx, defaultColNameOperation, bs.operationModels); err != nil {
		return err
	}

	if err := bs.writeModels(ctx, defaultColNameAccount, bs.accountModels); err != nil {
		return err
	}

	if err := bs.writeModels(ctx, defaultColNameBalance, bs.balanceModels); err != nil {
		return err
	}

	return nil
}

func (bs *BlockStorage) prepareOperationsTree() error {
	inStates := map[string]struct{}{}
	if err := bs.block.OperationsTree().Traverse(func(i int, key, _, v []byte) (bool, error) {
		fh := valuehash.NewBytes(key)

		switch mod, err := base.BytesToFactMode(v); {
		case err != nil:
			return false, err
		case mod&base.FInStates != 0:
			inStates[fh.String()] = struct{}{}
		}
		return true, nil
	}); err != nil {
		return err
	}

	bs.inStates = inStates

	return nil
}

func (bs *BlockStorage) prepareOperations() error {
	if len(bs.block.Operations()) < 1 {
		return nil
	}

	bs.operationModels = make([]mongo.WriteModel, len(bs.block.Operations()))

	inStates := func(valuehash.Hash) bool {
		return false
	}
	if bs.inStates != nil {
		inStates = func(fh valuehash.Hash) bool {
			_, found := bs.inStates[fh.String()]
			return found
		}
	}

	for i := range bs.block.Operations() {
		op := bs.block.Operations()[i]
		if doc, err := NewOperationDoc(
			op,
			bs.st.storage.Encoder(),
			bs.block.Height(),
			bs.block.ConfirmedAt(),
			inStates(op.Fact().Hash()),
		); err != nil {
			return err
		} else {
			bs.operationModels[i] = mongo.NewInsertOneModel().SetDocument(doc)
		}
	}

	return nil
}

func (bs *BlockStorage) prepareAccounts() error {
	if len(bs.block.States()) < 1 {
		return nil
	}

	var accountsModels []mongo.WriteModel
	var balancesModels []mongo.WriteModel
	for i := range bs.block.States() {
		st := bs.block.States()[i]
		if _, ok, err := IsAccountState(st); err != nil {
			return err
		} else if ok {
			if rs, err := NewAccountValue(st); err != nil {
				return err
			} else if doc, err := NewAccountDoc(rs, bs.st.storage.Encoder()); err != nil {
				return err
			} else {
				accountsModels = append(accountsModels, mongo.NewInsertOneModel().SetDocument(doc))
			}

			continue
		}

		if _, ok, err := IsBalanceState(st); err != nil {
			return err
		} else if ok {
			if doc, err := NewBalanceDoc(st, bs.st.storage.Encoder()); err != nil {
				return err
			} else {
				balancesModels = append(balancesModels, mongo.NewInsertOneModel().SetDocument(doc))
			}
		}
	}

	bs.accountModels = accountsModels
	bs.balanceModels = balancesModels

	return nil
}

func (bs *BlockStorage) writeModels(ctx context.Context, col string, models []mongo.WriteModel) error {
	started := time.Now()
	defer func() {
		bs.statesValue.Store(fmt.Sprintf("write-models-%s", col), time.Since(started))
	}()

	n := len(models)
	if n < 1 {
		return nil
	} else if n <= bulkWriteLimit {
		return bs.writeModelsChunk(ctx, col, models)
	}

	z := n / bulkWriteLimit
	if n%bulkWriteLimit != 0 {
		z++
	}

	for i := 0; i < z; i++ {
		s := i * bulkWriteLimit
		e := s + bulkWriteLimit
		if e > n {
			e = n
		}

		if err := bs.writeModelsChunk(ctx, col, models[s:e]); err != nil {
			return err
		}
	}

	return nil
}

func (bs *BlockStorage) writeModelsChunk(ctx context.Context, col string, models []mongo.WriteModel) error {
	opts := options.BulkWrite().SetOrdered(false)
	if res, err := bs.st.storage.Client().Collection(col).BulkWrite(ctx, models, opts); err != nil {
		return storage.WrapStorageError(err)
	} else if res != nil && res.InsertedCount < 1 {
		return xerrors.Errorf("not inserted to %s", col)
	}

	return nil
}

func (bs *BlockStorage) Close() error {
	bs.Lock()
	defer bs.Unlock()

	return bs.close()
}

func (bs *BlockStorage) close() error {
	bs.block = nil
	bs.operationModels = nil
	bs.accountModels = nil
	bs.balanceModels = nil

	return bs.st.Close()
}
