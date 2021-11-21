package cmds

import (
	"context"
	"net/url"

	"github.com/pkg/errors"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/launch/config"
	yamlconfig "github.com/spikeekips/mitum/launch/config/yaml"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"

	"github.com/spikeekips/mitum-currency/currency"
)

var (
	DefaultDigestAPICache *url.URL
	DefaultDigestAPIBind  string
	DefaultDigestAPIURL   string
)

func init() {
	DefaultDigestAPICache, _ = network.ParseURL("memory://", false)
	DefaultDigestAPIBind = "https://0.0.0.0:54320"
	DefaultDigestAPIURL = "https://127.0.0.1:54320"
}

type KeyDesign struct {
	PublickeyString string `yaml:"publickey"`
	Weight          uint
	Key             currency.Key `yaml:"-"`
}

func (kd *KeyDesign) IsValid([]byte) error {
	je, err := encs.Encoder(jsonenc.JSONEncoderType, "")
	if err != nil {
		return errors.Wrap(err, "json encoder needs for load design")
	}

	if pub, err := key.DecodePublickey(je, kd.PublickeyString); err != nil {
		return err
	} else if k, err := currency.NewKey(pub, kd.Weight); err != nil {
		return err
	} else {
		kd.Key = k
	}

	return nil
}

type AccountKeysDesign struct {
	Threshold  uint
	KeysDesign []*KeyDesign     `yaml:"keys"`
	Keys       currency.Keys    `yaml:"-"`
	Address    currency.Address `yaml:"-"`
}

func (akd *AccountKeysDesign) IsValid([]byte) error {
	ks := make([]currency.Key, len(akd.KeysDesign))
	for i := range akd.KeysDesign {
		kd := akd.KeysDesign[i]

		if err := kd.IsValid(nil); err != nil {
			return err
		}

		ks[i] = kd.Key
	}

	keys, err := currency.NewKeys(ks, akd.Threshold)
	if err != nil {
		return err
	}
	akd.Keys = keys

	a, err := currency.NewAddressFromKeys(akd.Keys)
	if err != nil {
		return err
	}
	akd.Address = a

	return nil
}

type GenesisCurrenciesDesign struct {
	AccountKeys *AccountKeysDesign `yaml:"account-keys"`
	Currencies  []*CurrencyDesign  `yaml:"currencies"`
}

func (de *GenesisCurrenciesDesign) IsValid([]byte) error {
	if de.AccountKeys == nil {
		return errors.Errorf("empty account-keys")
	}

	if err := de.AccountKeys.IsValid(nil); err != nil {
		return err
	}

	for i := range de.Currencies {
		if err := de.Currencies[i].IsValid(nil); err != nil {
			return err
		}
	}

	return nil
}

type CurrencyDesign struct {
	CurrencyString             *string         `yaml:"currency"`
	BalanceString              *string         `yaml:"balance"`
	NewAccountMinBalanceString *string         `yaml:"new-account-min-balance"`
	Feeer                      *FeeerDesign    `yaml:"feeer"`
	Balance                    currency.Amount `yaml:"-"`
	NewAccountMinBalance       currency.Big    `yaml:"-"`
}

func (de *CurrencyDesign) IsValid([]byte) error {
	var cid currency.CurrencyID
	if de.CurrencyString == nil {
		return errors.Errorf("empty currency")
	}
	cid = currency.CurrencyID(*de.CurrencyString)
	if err := cid.IsValid(nil); err != nil {
		return err
	}

	if de.BalanceString != nil {
		b, err := currency.NewBigFromString(*de.BalanceString)
		if err != nil {
			return err
		}
		de.Balance = currency.NewAmount(b, cid)
		if err := de.Balance.IsValid(nil); err != nil {
			return err
		}
	}

	if de.NewAccountMinBalanceString == nil {
		de.NewAccountMinBalance = currency.ZeroBig
	} else {
		b, err := currency.NewBigFromString(*de.NewAccountMinBalanceString)
		if err != nil {
			return err
		}
		de.NewAccountMinBalance = b
	}

	if de.Feeer == nil {
		de.Feeer = &FeeerDesign{}
	} else if err := de.Feeer.IsValid(nil); err != nil {
		return err
	}

	return nil
}

// FeeerDesign is used for genesis currencies and naturally it's receiver is genesis account
type FeeerDesign struct {
	Type   string
	Extras map[string]interface{} `yaml:",inline"`
}

func (no *FeeerDesign) IsValid([]byte) error {
	switch t := no.Type; t {
	case currency.FeeerNil, "":
	case currency.FeeerFixed:
		if err := no.checkFixed(no.Extras); err != nil {
			return err
		}
	case currency.FeeerRatio:
		if err := no.checkRatio(no.Extras); err != nil {
			return err
		}
	default:
		return errors.Errorf("unknown type of feeer, %v", t)
	}

	return nil
}

func (no FeeerDesign) checkFixed(c map[string]interface{}) error {
	a, found := c["amount"]
	if !found {
		return errors.Errorf("fixed needs `amount`")
	}
	n, err := currency.NewBigFromInterface(a)
	if err != nil {
		return errors.Wrapf(err, "invalid amount value, %v of fixed", a)
	}
	no.Extras["fixed_amount"] = n

	return nil
}

func (no FeeerDesign) checkRatio(c map[string]interface{}) error {
	if a, found := c["ratio"]; !found {
		return errors.Errorf("ratio needs `ratio`")
	} else if f, ok := a.(float64); !ok {
		return errors.Errorf("invalid ratio value type, %T of ratio; should be float64", a)
	} else {
		no.Extras["ratio_ratio"] = f
	}

	if a, found := c["min"]; !found {
		return errors.Errorf("ratio needs `min`")
	} else if n, err := currency.NewBigFromInterface(a); err != nil {
		return errors.Wrapf(err, "invalid min value, %v of ratio", a)
	} else {
		no.Extras["ratio_min"] = n
	}

	if a, found := c["max"]; found {
		n, err := currency.NewBigFromInterface(a)
		if err != nil {
			return errors.Wrapf(err, "invalid max value, %v of ratio", a)
		}
		no.Extras["ratio_max"] = n
	}

	return nil
}

type DigestDesign struct {
	NetworkYAML *yamlconfig.LocalNetwork `yaml:"network,omitempty"`
	CacheYAML   *string                  `yaml:"cache,omitempty"`
	network     config.LocalNetwork
	cache       *url.URL
}

func (no *DigestDesign) Set(ctx context.Context) (context.Context, error) {
	nctx := context.WithValue(
		context.Background(),
		config.ContextValueConfig,
		config.NewBaseLocalNode(nil, nil),
	)
	if no.NetworkYAML != nil {
		var conf config.LocalNode
		if i, err := no.NetworkYAML.Set(nctx); err != nil {
			return ctx, err
		} else if err := config.LoadConfigContextValue(i, &conf); err != nil {
			return ctx, err
		} else {
			no.network = conf.Network()
		}
	}

	var lconf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &lconf); err != nil {
		return ctx, err
	}

	if no.network.Bind() == nil {
		_ = no.network.SetBind(DefaultDigestAPIBind)
	}

	if no.network.ConnInfo() == nil {
		connInfo, _ := network.NewHTTPConnInfoFromString(DefaultDigestURL, lconf.Network().ConnInfo().Insecure())
		_ = no.network.SetConnInfo(connInfo)
	}

	if certs := no.network.Certs(); len(certs) < 1 {
		priv, err := util.GenerateED25519Privatekey()
		if err != nil {
			return ctx, err
		}

		host := "localhost"
		if no.network.ConnInfo() != nil {
			host = no.network.ConnInfo().URL().Hostname()
		}

		ct, err := util.GenerateTLSCerts(host, priv)
		if err != nil {
			return ctx, err
		}

		if err := no.network.SetCerts(ct); err != nil {
			return ctx, err
		}
	}

	if no.CacheYAML == nil {
		no.cache = DefaultDigestAPICache
	} else {
		u, err := network.ParseURL(*no.CacheYAML, true)
		if err != nil {
			return ctx, err
		}
		no.cache = u
	}

	return ctx, nil
}

func (no *DigestDesign) Network() config.LocalNetwork {
	return no.network
}

func (no *DigestDesign) Cache() *url.URL {
	return no.cache
}
