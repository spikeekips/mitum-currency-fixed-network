package digest

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type DigestError struct {
	err    error
	height base.Height
}

func NewDigestError(err error, height base.Height) DigestError {
	if err == nil {
		return DigestError{height: height}
	}

	return DigestError{err: err, height: height}
}

func (de DigestError) Error() string {
	if de.err == nil {
		return ""
	}

	return de.err.Error()
}

func (de DigestError) Height() base.Height {
	return de.height
}

func (de DigestError) IsError() bool {
	return de.err != nil
}

type Digester struct {
	sync.RWMutex
	*util.FunctionDaemon
	*logging.Logging
	storage   *Storage
	blockChan chan block.Block
	errChan   chan error
}

func NewDigester(st *Storage, errChan chan error) *Digester {
	di := &Digester{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "digester")
		}),
		storage:   st,
		blockChan: make(chan block.Block, 100),
		errChan:   errChan,
	}

	di.FunctionDaemon = util.NewFunctionDaemon(di.start, false)

	return di
}

func (di *Digester) start(stopchan chan struct{}) error {
end:
	for {
		select {
		case <-stopchan:
			di.Log().Debug().Msg("stopped")

			break end
		case blk := <-di.blockChan:
			go func(blk block.Block) {
				if err := util.Retry(0, time.Second*3, func() error {
					if err := di.digest(blk); err != nil {
						if di.errChan != nil {
							di.errChan <- NewDigestError(err, blk.Height())
						}

						return err
					}

					return nil
				}); err != nil {
					di.Log().Error().Err(err).Hinted("block", blk.Height()).Msg("failed to digest block")
				} else {
					di.Log().Info().Hinted("block", blk.Height()).Msg("block digested")

					if di.errChan != nil {
						di.errChan <- NewDigestError(nil, blk.Height())
					}
				}
			}(blk)
		}
	}

	return nil
}

func (di *Digester) Digest(blocks []block.Block) {
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Height() < blocks[j].Height()
	})

	for i := range blocks {
		blk := blocks[i]
		di.Log().Debug().Hinted("block", blk.Height()).Msg("start to digest block")

		di.blockChan <- blk
	}
}

func (di *Digester) digest(blk block.Block) error {
	di.Lock()
	defer di.Unlock()

	return DigestBlock(di.storage, blk)
}

func DigestBlock(st *Storage, blk block.Block) error {
	var bs *BlockStorage
	if s, err := NewBlockStorage(st, blk); err != nil {
		return err
	} else {
		bs = s

		defer func() {
			_ = bs.Close()
		}()
	}

	if err := bs.Prepare(); err != nil {
		return err
	} else if err := bs.Commit(context.Background()); err != nil {
		return err
	} else {
		return st.SetLastBlock(blk.Height())
	}
}
