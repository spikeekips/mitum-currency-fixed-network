package currency

import (
	"testing"
	"time"

	"github.com/spikeekips/mitum/isaac"
	"github.com/stretchr/testify/suite"
)

type testConcurrentOperationsProcessor struct {
	suite.Suite
	isaac.StorageSupportTest
}

func (t *testConcurrentOperationsProcessor) TestNew() {
	sp, err := isaac.NewStatepool(t.Storage(nil, nil))
	t.NoError(err)

	{
		_, err := NewConcurrentOperationsProcessor(0, sp, time.Second, nil)
		t.Contains(err.Error(), "size must be over 0")
	}

	{
		co, err := NewConcurrentOperationsProcessor(maxConcurrentOperations+1, sp, time.Second, nil)
		t.NoError(err)
		t.Equal(maxConcurrentOperations, co.size)
	}

	{
		co, err := NewConcurrentOperationsProcessor(1, sp, time.Second, nil)
		t.NoError(err)
		t.NoError(co.Close())
	}

	{
		co, err := NewConcurrentOperationsProcessor(1, sp, time.Second, nil)
		t.NoError(err)

		co.Start()
		t.NoError(co.Close())
	}
}

func TestConcurrentOperationsProcessor(t *testing.T) {
	suite.Run(t, new(testConcurrentOperationsProcessor))
}
