package currency

import (
	"strings"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/policy"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func LoadPolicyOperation(design *launcher.NodeDesign) ([]operation.Operation, error) {
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
	Balance       Amount             `yaml:"-"`
}

func (gad *GenesisAccountDesign) IsValid([]byte) error {
	gad.AccountKeys.encs = gad.encs
	if err := gad.AccountKeys.IsValid(nil); err != nil {
		return err
	}

	if am, err := NewAmountFromString(gad.BalanceString); err != nil {
		return err
	} else {
		gad.Balance = am
	}

	return nil
}

type AccountKeysDesign struct {
	encs       *encoder.Encoders
	Threshold  uint
	KeysDesign []*KeyDesign `yaml:"keys"`
	Keys       Keys         `yaml:"-"`
	Address    Address      `yaml:"-"`
}

func (akd *AccountKeysDesign) IsValid([]byte) error {
	ks := make([]Key, len(akd.KeysDesign))
	for i := range akd.KeysDesign {
		kd := akd.KeysDesign[i]
		kd.encs = akd.encs

		if err := kd.IsValid(nil); err != nil {
			return err
		}

		ks[i] = kd.Key
	}

	if keys, err := NewKeys(ks, akd.Threshold); err != nil {
		return err
	} else {
		akd.Keys = keys
	}

	if a, err := NewAddressFromKeys(akd.Keys); err != nil {
		return err
	} else {
		akd.Address = a
	}

	return nil
}

type KeyDesign struct {
	encs             *encoder.Encoders
	PrivatekeyString string `yaml:"privatekey"`
	Weight           uint
	Key              Key `yaml:"-"`
}

func (kd *KeyDesign) IsValid([]byte) error {
	var je encoder.Encoder
	if e, err := kd.encs.Encoder(jsonenc.JSONType, ""); err != nil {
		return xerrors.Errorf("json encoder needs for load design: %w", err)
	} else {
		je = e
	}

	if pk, err := key.DecodePrivatekey(je, kd.PrivatekeyString); err != nil {
		return err
	} else {
		k := NewKey(pk.Publickey(), kd.Weight)
		if err := k.IsValid(nil); err != nil {
			return err
		}

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

func LoadGenesisAccountOperation(nr *Launcher, m map[string]interface{}) (GenesisAccount, error) {
	var gad *GenesisAccountDesign
	if d, err := LoadGenesisAccountDesign(nr, m); err != nil {
		return GenesisAccount{}, err
	} else {
		gad = d
	}

	if op, err := NewGenesisAccount(
		nr.Design().Privatekey(),
		gad.AccountKeys.Keys,
		gad.Balance,
		nr.Design().NetworkID(),
	); err != nil {
		return GenesisAccount{}, err
	} else {
		return op, nil
	}
}
