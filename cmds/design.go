package cmds

import (
	"strings"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/policy"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type NodeDesign struct {
	*launcher.NodeDesign
	FeeAmount   currency.FeeAmount
	FeeReceiver base.Address
}

func (nd *NodeDesign) IsValid(b []byte) error {
	if err := nd.NodeDesign.IsValid(b); err != nil {
		return err
	}

	if err := nd.loadFeeAmount(); err != nil {
		return err
	}

	return nil
}

func (nd *NodeDesign) loadFeeAmount() error {
	var c map[string]interface{}
	if nd.Component.Others() != nil {
		for k, v := range nd.Component.Others() {
			if k != "fee-amount" {
				continue
			}

			if v == nil {
				continue
			} else if m, ok := v.(map[string]interface{}); !ok {
				return xerrors.Errorf("bad formatted fee-amount design")
			} else {
				c = m
			}
		}
	}

	if c == nil {
		return nil
	}

	if i, found := c["to"]; found {
		if s, ok := i.(string); !ok {
			return xerrors.Errorf("invalid type, %T of to of fee-amount", i)
		} else if a, err := base.DecodeAddressFromString(nd.JSONEncoder(), strings.TrimSpace(s)); err != nil {
			return err
		} else if err := a.IsValid(nil); err != nil {
			return err
		} else {
			nd.FeeReceiver = a
		}
	}

	var fa currency.FeeAmount
	switch t := c["type"]; {
	case t == "fixed":
		if f, err := nd.loadFixedFeeAmount(c); err != nil {
			return err
		} else {
			fa = f
		}
	case t == "ratio":
		if f, err := nd.loadRatioFeeAmount(c); err != nil {
			return err
		} else {
			fa = f
		}
	default:
		return xerrors.Errorf("unknown type of fee-amount, %v", t)
	}

	nd.FeeAmount = fa

	return nil
}

func (nd *NodeDesign) loadFixedFeeAmount(c map[string]interface{}) (currency.FeeAmount, error) {
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

func (nd *NodeDesign) loadRatioFeeAmount(c map[string]interface{}) (currency.FeeAmount, error) {
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

func LoadNodeDesign(b []byte, encs *encoder.Encoders) (*NodeDesign, error) {
	if d, err := launcher.LoadNodeDesign(b, encs); err != nil {
		return nil, err
	} else {
		nd := &NodeDesign{NodeDesign: d}
		if err := nd.IsValid(nil); err != nil {
			return nil, err
		}

		return nd, nil
	}
}

func LoadPolicyOperation(design *NodeDesign) ([]operation.Operation, error) {
	if op, err := policy.NewSetPolicyV0(
		design.GenesisPolicy.Policy().(policy.PolicyV0),
		design.NetworkID(), // NOTE token
		design.Privatekey(),
		design.NetworkID(),
	); err != nil {
		return nil, xerrors.Errorf("failed to create SetPolicyOperation: %w", err)
	} else {
		return []operation.Operation{op}, nil
	}
}

type GenesisAccountDesign struct {
	encs          *encoder.Encoders
	AccountKeys   *AccountKeysDesign `yaml:"account-keys"`
	BalanceString string             `yaml:"balance"`
	Balance       currency.Amount    `yaml:"-"`
}

func (gad *GenesisAccountDesign) IsValid([]byte) error {
	gad.AccountKeys.encs = gad.encs
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

type AccountKeysDesign struct {
	encs       *encoder.Encoders
	Threshold  uint
	KeysDesign []*KeyDesign     `yaml:"keys"`
	Keys       currency.Keys    `yaml:"-"`
	Address    currency.Address `yaml:"-"`
}

func (akd *AccountKeysDesign) IsValid([]byte) error {
	ks := make([]currency.Key, len(akd.KeysDesign))
	for i := range akd.KeysDesign {
		kd := akd.KeysDesign[i]
		kd.encs = akd.encs

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

type KeyDesign struct {
	encs            *encoder.Encoders
	PublickeyString string `yaml:"publickey"`
	Weight          uint
	Key             currency.Key `yaml:"-"`
}

func (kd *KeyDesign) IsValid([]byte) error {
	var je encoder.Encoder
	if e, err := kd.encs.Encoder(jsonenc.JSONType, ""); err != nil {
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

func LoadGenesisAccountDesign(
	nr *Launcher,
	m map[string]interface{},
) (*GenesisAccountDesign, error) {
	var gad *GenesisAccountDesign
	if b, err := yaml.Marshal(m); err != nil {
		return nil, err
	} else if err := yaml.Unmarshal(b, &gad); err != nil {
		return nil, err
	}

	gad.encs = nr.Encoders()
	if err := gad.IsValid(nil); err != nil {
		return nil, err
	}

	return gad, nil
}

func LoadOtherInitOperations(nr *Launcher) ([]operation.Operation, error) {
	var ops []operation.Operation
	for i := range nr.Design().InitOperations {
		m := nr.Design().InitOperations[i]

		if name, found := m["name"]; !found {
			return nil, xerrors.Errorf("invalid format found")
		} else if len(strings.TrimSpace(name.(string))) < 1 {
			return nil, xerrors.Errorf("invalid format found; empty name")
		} else if op, err := LoadOtherInitOperation(nr, name.(string), m); err != nil {
			return nil, err
		} else {
			ops = append(ops, op)
		}
	}

	return ops, nil
}

func LoadOtherInitOperation(nr *Launcher, name string, m map[string]interface{}) (operation.Operation, error) {
	switch name {
	case "genesis-account":
		return LoadGenesisAccountOperation(nr, m)
	default:
		return nil, xerrors.Errorf("unknown operation name found, %q", name)
	}
}

func LoadGenesisAccountOperation(nr *Launcher, m map[string]interface{}) (currency.GenesisAccount, error) {
	var gad *GenesisAccountDesign
	if d, err := LoadGenesisAccountDesign(nr, m); err != nil {
		return currency.GenesisAccount{}, err
	} else {
		gad = d
	}

	if op, err := currency.NewGenesisAccount(
		nr.Design().Privatekey(),
		gad.AccountKeys.Keys,
		gad.Balance,
		nr.Design().NetworkID(),
	); err != nil {
		return currency.GenesisAccount{}, err
	} else {
		return op, nil
	}
}
