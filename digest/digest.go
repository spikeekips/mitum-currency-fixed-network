package digest

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
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
	*util.ContextDaemon
	*logging.Logging
	database  *Database
	blockChan chan block.Block
	errChan   chan error
}

func NewDigester(st *Database, errChan chan error) *Digester {
	di := &Digester{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "digester")
		}),
		database:  st,
		blockChan: make(chan block.Block, 100),
		errChan:   errChan,
	}

	di.ContextDaemon = util.NewContextDaemon("digester", di.start)

	return di
}

func (di *Digester) start(ctx context.Context) error {
	errch := func(err DigestError) {
		if di.errChan == nil {
			return
		}

		di.errChan <- err
	}

end:
	for {
		select {
		case <-ctx.Done():
			di.Log().Debug().Msg("stopped")

			break end
		case blk := <-di.blockChan:
			err := util.Retry(0, time.Second*1, func(int) error {
				if err := di.digest(ctx, blk); err != nil {
					go errch(NewDigestError(err, blk.Height()))

					if errors.Is(err, context.Canceled) {
						return util.StopRetryingError.Wrap(err)
					}

					return err
				}

				return nil
			})
			if err != nil {
				di.Log().Error().Err(err).Int64("block", blk.Height().Int64()).Msg("failed to digest block")
			} else {
				di.Log().Info().Int64("block", blk.Height().Int64()).Msg("block digested")
			}

			go errch(NewDigestError(err, blk.Height()))
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
		di.Log().Debug().Int64("block", blk.Height().Int64()).Msg("start to digest block")

		di.blockChan <- blk
	}
}

func (di *Digester) digest(ctx context.Context, blk block.Block) error {
	di.Lock()
	defer di.Unlock()

	if err := DigestBlock(ctx, di.database, blk); err != nil {
		return err
	}

	return di.database.SetLastBlock(blk.Height())
}

func DigestBlock(ctx context.Context, st *Database, blk block.Block) error {
	bs, err := NewBlockSession(st, blk)
	if err != nil {
		return err
	}
	defer func() {
		_ = bs.Close()
	}()

	if err := bs.Prepare(); err != nil {
		return err
	}

	return bs.Commit(ctx)
}
