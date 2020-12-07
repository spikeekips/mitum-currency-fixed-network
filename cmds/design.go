package cmds

import (
	"context"
	"net/url"
	"time"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/launch/config"
	yamlconfig "github.com/spikeekips/mitum/launch/config/yaml"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"

	"github.com/spikeekips/mitum-currency/currency"
)

var (
	DefaultDigestAPICache *url.URL
	DefaultDigestAPIBind  string
	DefaultDigestAPIURL   string
)

func init() {
	DefaultDigestAPICache, _ = url.Parse("memory://")
	DefaultDigestAPIBind = "https://0.0.0.0:54320"
	DefaultDigestAPIURL = "https://127.0.0.1:54320"
}

type KeyDesign struct {
	PublickeyString string `yaml:"publickey"`
	Weight          uint
	Key             currency.Key `yaml:"-"`
}

func (kd *KeyDesign) IsValid([]byte) error {
	var je encoder.Encoder
	if e, err := encs.Encoder(jsonenc.JSONType, ""); err != nil {
		return xerrors.Errorf("json encoder needs for load design: %w", err)
	} else {
		je = e
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

	if keys, err := currency.NewKeys(ks, akd.Threshold); err != nil {
		return err
	} else {
		akd.Keys = keys
	}

	if a, err := currency.NewAddressFromKeys(akd.Keys); err != nil {
		return err
	} else {
		akd.Address = a
	}

	return nil
}

type GenesisAccountDesign struct {
	AccountKeys   *AccountKeysDesign `yaml:"account-keys"`
	BalanceString string             `yaml:"balance"`
	Balance       currency.Amount    `yaml:"-"`
}

func (gad *GenesisAccountDesign) IsValid([]byte) error {
	if err := gad.AccountKeys.IsValid(nil); err != nil {
		return err
	}

	if am, err := currency.NewAmountFromString(gad.BalanceString); err != nil {
		return err
	} else {
		gad.Balance = am
	}

	return nil
}

type FeeDesign struct {
	Type           string
	ReceiverString string             `yaml:"receiver,omitempty"`
	Receiver       base.Address       `yaml:"-"`
	FeeAmount      currency.FeeAmount `yaml:"-"`
	ReceiverFunc   func() (base.Address, error)
	extras         map[string]interface{}
}

func (no *FeeDesign) UnmarshalYAML(value *yaml.Node) error {
	var m struct {
		Type     string
		Receiver string
		Extras   map[string]interface{} `yaml:",inline"`
	}

	if err := value.Decode(&m); err != nil {
		return err
	}

	var fa currency.FeeAmount
	switch t := m.Type; t {
	case "":
		fa = currency.NewNilFeeAmount()
	case "fixed":
		if f, err := no.loadFixedFeeAmount(m.Extras); err != nil {
			return err
		} else {
			fa = f
		}
	case "ratio":
		if f, err := no.loadRatioFeeAmount(m.Extras); err != nil {
			return err
		} else {
			fa = f
		}
	default:
		return xerrors.Errorf("unknown type of fee-amount, %v", t)
	}

	no.Type = m.Type
	no.ReceiverString = m.Receiver
	no.FeeAmount = fa
	no.extras = m.Extras

	return nil
}

func (no FeeDesign) loadFixedFeeAmount(c map[string]interface{}) (currency.FeeAmount, error) {
	if a, found := c["amount"]; !found {
		return nil, xerrors.Errorf("fixed fee-amount needs `amount`")
	} else {
		if n, err := currency.NewAmountFromInterface(a); err != nil {
			return nil, xerrors.Errorf("invalid amount value, %v of fee-amount: %w", a, err)
		} else {
			return currency.NewFixedFeeAmount(n), nil
		}
	}
}

func (no FeeDesign) loadRatioFeeAmount(c map[string]interface{}) (currency.FeeAmount, error) {
	var ratio float64
	if a, found := c["ratio"]; !found {
		return nil, xerrors.Errorf("ratio fee-amount needs `ratio`")
	} else if f, ok := a.(float64); !ok {
		return nil, xerrors.Errorf("invalid ratio value type, %T of fee-amount; should be float64", a)
	} else {
		ratio = f
	}

	var min currency.Amount
	if a, found := c["min"]; !found {
		return nil, xerrors.Errorf("ratio fee-amount needs `min`")
	} else if n, err := currency.NewAmountFromInterface(a); err != nil {
		return nil, xerrors.Errorf("invalid min value, %v of fee-amount: %w", a, err)
	} else {
		min = n
	}

	return currency.NewRatioFeeAmount(ratio, min)
}

type DigestDesign struct {
	NetworkYAML     *yamlconfig.LocalNetwork `yaml:"network,omitempty"`
	CacheYAML       *string                  `yaml:"cache,omitempty"`
	RateLimiterYAML *RateLimiterDesign       `yaml:"rate-limit"`
	network         config.LocalNetwork
	cache           *url.URL
	rateLimiter     *limiter.Limiter
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
	if no.network.Bind() == nil {
		_ = no.network.SetBind(DefaultDigestAPIBind)
	}
	if no.network.URL() == nil {
		_ = no.network.SetURL(DefaultDigestAPIURL)
	}

	if no.CacheYAML == nil {
		no.cache = DefaultDigestAPICache
	} else {
		if u, err := config.ParseURLString(*no.CacheYAML, true); err != nil {
			return ctx, err
		} else {
			no.cache = u
		}
	}

	if no.RateLimiterYAML != nil {
		if err := no.RateLimiterYAML.Set(ctx); err != nil {
			return ctx, err
		} else {
			no.rateLimiter = no.RateLimiterYAML.Limiter()
		}
	}

	return ctx, nil
}

func (no *DigestDesign) Network() config.LocalNetwork {
	return no.network
}

func (no *DigestDesign) Cache() *url.URL {
	return no.cache
}

func (no *DigestDesign) RateLimiter() *limiter.Limiter {
	return no.rateLimiter
}

type RateLimiterDesign struct {
	PeriodYAML *string `yaml:"period"`
	Limit      *uint64
	limiter    *limiter.Limiter
}

func (no *RateLimiterDesign) Set(context.Context) error {
	if no.PeriodYAML == nil {
		return xerrors.Errorf("period is missing")
	} else {
		var period time.Duration
		switch d, err := time.ParseDuration(*no.PeriodYAML); {
		case err != nil:
			return xerrors.Errorf("invalid period string, %q: %w", no.PeriodYAML, err)
		case d < 0:
			return xerrors.Errorf("negative period string, %q", no.PeriodYAML)
		default:
			period = d
		}

		if no.Limit == nil || *no.Limit < 1 {
			return xerrors.Errorf("limit should be over 0")
		}

		no.limiter = limiter.New(
			memory.NewStore(),
			limiter.Rate{Period: period, Limit: int64(*no.Limit)},
			limiter.WithTrustForwardHeader(true),
		)
	}

	return nil
}

func (no RateLimiterDesign) Limiter() *limiter.Limiter {
	return no.limiter
}
