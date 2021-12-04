package cmds

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type testGenesisCurrencies struct {
	suite.Suite
}

func (t *testGenesisCurrencies) TestLoad() {
	encs := encoder.NewEncoders()
	encs.TestAddHinter(key.BTCPrivatekeyHinter)
	encs.TestAddHinter(key.BTCPublickeyHinter)

	enc := jsonenc.NewEncoder()
	encs.AddEncoder(enc)

	conf := config.NewBaseLocalNode(enc, nil)

	pub := key.MustNewBTCPrivatekey().Publickey()

	k, err := currency.NewBaseAccountKey(pub, 99)
	t.NoError(err)
	ks, err := currency.NewBaseAccountKeys([]currency.AccountKey{k}, 98)
	t.NoError(err)

	genesisAccount, err := currency.NewAddressFromKeys(ks)
	t.NoError(err)

	t.NoError(conf.SetPrivatekey(key.MustNewBTCPrivatekey().String()))
	t.NoError(conf.SetNetworkID("Fri 29 Jan 2001 12:00:02 AM KST"))

	ctx := context.WithValue(context.Background(), config.ContextValueConfig, conf)

	y := fmt.Sprintf(`
account-keys:
  keys:
    - publickey: %s
      weight: 99
  threshold: 98

currencies:
  - currency: SHOW*ME
    balance: "9999999999999999999999999999999999"
    new-account-min-balance: "33"
    feeer:
      type: fixed
      amount: 33
`, pub.String())

	var m map[string]interface{}
	t.NoError(yaml.Unmarshal([]byte(y), &m))

	op, err := GenesisOperationsHandlerGenesisCurrencies(ctx, m)
	t.NoError(err)
	t.NotNil(op)

	t.NoError(op.IsValid(conf.NetworkID()))

	fact := op.Fact().(currency.GenesisCurrenciesFact)

	t.True(conf.Privatekey().Publickey().Equal(fact.GenesisNodeKey()))
	t.Equal(1, len(fact.Keys().Keys()))
	t.Equal(uint(98), fact.Keys().Threshold())
	t.True(pub.Equal(fact.Keys().Keys()[0].Key()))
	t.Equal(uint(99), fact.Keys().Keys()[0].Weight())

	t.Equal(1, len(fact.Currencies()))
	t.Equal(currency.CurrencyID("SHOW*ME"), fact.Currencies()[0].Currency())
	t.Nil(fact.Currencies()[0].GenesisAccount())
	t.Equal("9999999999999999999999999999999999", fact.Currencies()[0].Big().String())

	feeer := fact.Currencies()[0].Policy().Feeer()
	t.Equal(currency.FixedFeeerType, feeer.Hint().Type())
	t.Equal("33", feeer.Min().String())
	t.True(genesisAccount.Equal(feeer.Receiver()))
}

func TestGenesisCurrencies(t *testing.T) {
	suite.Run(t, new(testGenesisCurrencies))
}
