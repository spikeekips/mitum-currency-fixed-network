package currency

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type testAmountStat struct {
	baseTest
}

func (t *testAmountStat) TestNew() {
	address, err := NewAddress(util.UUID().String())
	t.NoError(err)

	amount := NewAmount(10)
	fee := NewAmount(3)

	key := StateKeyBalance(address)
	value, _ := state.NewStringValue(amount.String())

	sst, err := state.NewStateV0(key, value, base.NilHeight)
	t.NoError(err)

	st := NewAmountState(sst)
	st = st.AddFee(fee)

	_, ok := (interface{})(st).(state.State)
	t.True(ok)
}

func (t *testAmountStat) TestBSON() {
	encs := encoder.NewEncoders()
	benc := bsonenc.NewEncoder()
	jenc := jsonenc.NewEncoder()

	t.NoError(encs.AddEncoder(benc))
	t.NoError(encs.AddEncoder(jenc))

	_ = encs.AddHinter(state.BytesValue{})
	_ = encs.AddHinter(state.DurationValue{})
	_ = encs.AddHinter(state.HintedValue{})
	_ = encs.AddHinter(state.NumberValue{})
	_ = encs.AddHinter(state.SliceValue{})
	_ = encs.AddHinter(state.StateV0{})
	_ = encs.AddHinter(state.StringValue{})
	_ = encs.AddHinter(Key{})
	_ = encs.AddHinter(Keys{})
	_ = encs.AddHinter(Address(""))
	_ = encs.AddHinter(Account{})
	_ = encs.AddHinter(AmountState{})

	address, err := NewAddress(util.UUID().String())
	t.NoError(err)

	amount := NewAmount(10)
	fee := NewAmount(3)

	key := StateKeyBalance(address)
	value, _ := state.NewStringValue(amount.String())

	sst, err := state.NewStateV0(key, value, base.NilHeight)
	t.NoError(err)

	st := NewAmountState(sst)
	st = st.AddFee(fee)

	osst := (interface{})(st).(state.State)
	osst, err = osst.SetHash(osst.GenerateHash())
	t.NoError(err)
	osst = osst.SetHeight(base.Height(33))
	osst, err = osst.SetPreviousHeight(base.Height(32))
	t.NoError(err)

	st = osst.(AmountState)

	{
		b, err := bsonenc.Marshal(st)
		t.NoError(err)

		usst, err := state.DecodeState(benc, b)
		t.NoError(err)

		ust, ok := usst.(AmountState)
		t.True(ok)

		t.compare(st, ust)
	}

	{
		b, err := jsonenc.Marshal(st)
		t.NoError(err)

		usst, err := state.DecodeState(jenc, b)
		t.NoError(err)

		ust, ok := usst.(AmountState)
		t.True(ok)

		t.compare(st, ust)
	}
}

func (t *testAmountStat) compare(a, b AmountState) {
	t.True(a.Hint().Equal(b.Hint()))
	t.Equal(a.Key(), b.Key())
	t.True(a.Value().Equal(b.Value()))
	t.True(a.Hash().Equal(b.Hash()))
	t.True(a.Fee().Equal(b.Fee()))
}

func TestAmountStat(t *testing.T) {
	suite.Run(t, new(testAmountStat))
}
