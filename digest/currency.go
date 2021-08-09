package digest

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/state"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

func LoadCurrenciesFromDatabase(
	st *mongodbstorage.Database,
	height base.Height,
	callback func(state.State) (bool, error),
) error {
	var keys []string
	for {
		filter := util.NewBSONFilter("key", bson.M{
			"$regex": fmt.Sprintf(`^%s`, regexp.QuoteMeta(currency.StateKeyCurrencyDesignPrefix)),
		}).Add("height", bson.M{"$gte": height})

		var q primitive.D
		if len(keys) < 1 {
			q = filter.D()
		} else {
			q = filter.Add("key", bson.M{"$nin": keys}).D()
		}

		var sta state.State
		if err := st.Client().GetByFilter(mongodbstorage.ColNameState, q,
			func(res *mongo.SingleResult) error {
				i, err := loadStateFromDecoder(res.Decode, st.Encoders())
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

			return errors.Wrap(err, "failed to get currency state")
		}

		switch keep, err := callback(sta); {
		case err != nil:
			return err
		case !keep:
			return nil
		default:
			keys = append(keys, sta.Key())
		}
	}

	return nil
}

func loadStateFromDecoder(decoder func(interface{}) error, encs *encoder.Encoders) (state.State, error) {
	var b bson.Raw
	if err := decoder(&b); err != nil {
		return nil, err
	}

	var st state.State

	_, hinter, err := mongodbstorage.LoadDataFromDoc(b, encs)
	if err != nil {
		return nil, err
	} else if i, ok := hinter.(state.State); !ok {
		return nil, errors.Errorf("not state.State: %T", hinter)
	} else {
		st = i
	}

	return st, nil
}
