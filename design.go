package mc

import (
	"io/ioutil"
	"path/filepath"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/util/encoder"
)

func LoadDesign(f string, encs *encoder.Encoders) (*NodeDesign, error) {
	if d, err := LoadNodeDesignFromFile(f, encs); err != nil {
		return nil, xerrors.Errorf("failed to load design file: %w", err)
	} else if err := d.IsValid(nil); err != nil {
		return nil, xerrors.Errorf("invalid design file: %w", err)
	} else {
		return d, nil
	}
}

type NodeDesign struct {
	*launcher.NodeDesign
}

func LoadNodeDesignFromFile(f string, encs *encoder.Encoders) (*NodeDesign, error) {
	var design *NodeDesign
	if b, err := ioutil.ReadFile(filepath.Clean(f)); err != nil {
		return nil, err
	} else if err := yaml.Unmarshal(b, &design); err != nil {
		return nil, err
	}

	design.SetEncoders(encs)

	return design, nil
}

func (nd NodeDesign) MarshalYAML() (interface{}, error) {
	return nd.NodeDesign, nil
}

func (nd *NodeDesign) UnmarshalYAML(value *yaml.Node) error {
	var d *launcher.NodeDesign
	if err := value.Decode(&d); err != nil {
		return err
	}

	*nd = NodeDesign{NodeDesign: d}

	return nil
}
