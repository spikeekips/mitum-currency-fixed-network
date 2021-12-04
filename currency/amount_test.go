package currency

import (
	"encoding/json"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
)

type testAmount struct {
	suite.Suite
}

func (t *testAmount) TestWithBig() {
	cid := CurrencyID("SHOWME")

	a := MustNewAmount(NewBig(33), cid)
	t.Equal(a, a.WithBig(NewBig(33)))

	_ = a.WithBig(NewBig(44))

	t.Equal(a.Big(), NewBig(33))
}

func TestAmount(t *testing.T) {
	suite.Run(t, new(testAmount))
}

func testAmountEncode(enc encoder.Encoder) suite.TestingSuite {
	t := new(baseTestEncode)

	t.enc = enc
	t.newObject = func() interface{} {
		cid := CurrencyID("FINDME")

		am := NewAmount(NewBig(99), cid)
		t.NoError(am.IsValid(nil))

		am.BaseHinter = hint.NewBaseHinter(hint.NewHint(AmountType, "v0.0.9"))

		return am
	}

	t.encode = func(enc encoder.Encoder, v interface{}) ([]byte, error) {
		b, err := enc.Marshal(struct {
			A Amount
		}{A: v.(Amount)})
		if err != nil {
			return nil, err
		}

		switch enc.Hint().Type() {
		case jsonenc.JSONEncoderType:
			var D struct {
				A json.RawMessage
			}
			if err := enc.Unmarshal(b, &D); err != nil {
				return nil, err
			} else {
				return []byte(D.A), nil
			}
		case bsonenc.BSONEncoderType:
			var D struct {
				A bson.Raw
			}
			if err := enc.Unmarshal(b, &D); err != nil {
				return nil, err
			} else {
				return []byte(D.A), nil
			}
		default:
			return nil, errors.Errorf("unknown encoder, %v", enc)
		}
	}

	t.decode = func(enc encoder.Encoder, b []byte) (interface{}, error) {
		return enc.Decode(b)
	}

	t.compare = func(a, b interface{}) {
		ca := a.(Amount)
		cb := b.(Amount)

		t.True(ca.Hint().Equal(cb.Hint()))
		t.True(ca.Big().Equal(cb.Big()))
		t.Equal(ca.Currency(), cb.Currency())
	}

	return t
}

func TestAmountEncodeJSON(t *testing.T) {
	suite.Run(t, testAmountEncode(jsonenc.NewEncoder()))
}

func TestAmountEncodeBSON(t *testing.T) {
	suite.Run(t, testAmountEncode(bsonenc.NewEncoder()))
}
