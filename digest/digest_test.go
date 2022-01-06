//go:build mongodb
// +build mongodb

package digest

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/stretchr/testify/suite"
)

type testDigester struct {
	baseTest
}

func (t *testDigester) TestNew() {
	st, err := NewDatabase(t.MongodbDatabase(), t.MongodbDatabase())
	t.NoError(err)

	di := NewDigester(st, nil)
	t.NotNil(di)
}

func (t *testDigester) TestDigest() {
	mst := t.MongodbDatabase()
	st, err := NewDatabase(mst, t.MongodbDatabase())
	t.NoError(err)

	target := base.Height(3)

	var blocks []block.Block
	for i := base.GenesisHeight; i <= target; i++ {
		blk := t.newBlock(i, mst)
		blocks = append(blocks, blk)
	}

	errChan := make(chan error, 100)
	di := NewDigester(st, errChan)
	t.NotNil(di)

	t.NoError(di.Start())
	defer di.Stop()

	di.Digest(blocks)

	var digested []base.Height

end:
	for {
		select {
		case <-time.After(time.Second * 10):
			t.NoError(errors.Errorf("timeout to digest"))

			break end
		case err := <-errChan:
			switch i := err.(type) {
			case DigestError:
				if !i.IsError() {
					digested = append(digested, i.height)
				}
			default:
				t.NoError(err)

				break end
			}

			if len(digested) == len(blocks) {
				break end
			}
		}
	}
	t.Equal(target, st.LastBlock())
}

func (t *testDigester) TestDigestAgain() {
	mst := t.MongodbDatabase()
	st, err := NewDatabase(mst, t.MongodbDatabase())
	t.NoError(err)

	target := base.Height(3)

	var blocks []block.Block
	for i := base.GenesisHeight; i <= target; i++ {
		blk := t.newBlock(i, mst)
		blocks = append(blocks, blk)
	}

	errChan := make(chan error, 100)
	di := NewDigester(st, errChan)
	t.NotNil(di)

	t.NoError(di.Start())
	defer di.Stop()

	di.Digest(blocks)
	di.Digest(blocks[len(blocks)-2:])

	var digested []base.Height

end:
	for {
		select {
		case <-time.After(time.Second * 10):
			t.NoError(errors.Errorf("timeout to digest"))

			break end
		case err := <-errChan:
			switch i := err.(type) {
			case DigestError:
				if !i.IsError() {
					digested = append(digested, i.height)
				}
			default:
				t.NoError(err)

				break end
			}

			if len(digested) == len(blocks)+1 {
				break end
			}
		}
	}
	t.Equal(target, st.LastBlock())
}

func TestDigester(t *testing.T) {
	suite.Run(t, new(testDigester))
}
