package cmds

import (
	"fmt"
	"os"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/key"
)

type GenerateKeyCommand struct {
	Type   string `name:"type" help:"key type {btc ether stellar} (default: btc)" optional:"" default:"btc"`
	JSON   bool   `name:"json" help:"json output format (default: false)" optional:"" default:"false"`
	Pretty bool   `name:"pretty" help:"pretty format"`
}

func (cmd *GenerateKeyCommand) Run() error {
	if len(cmd.Type) < 1 {
		cmd.Type = btc
	} else {
		switch cmd.Type {
		case btc, ether, stellar:
		default:
			return xerrors.Errorf("unknown key type, %q", cmd.Type)
		}
	}

	var priv key.Privatekey
	switch cmd.Type {
	case btc:
		priv = key.MustNewBTCPrivatekey()
	case ether:
		priv = key.MustNewEtherPrivatekey()
	case stellar:
		priv = key.MustNewStellarPrivatekey()
	}

	if cmd.JSON {
		prettyPrint(cmd.Pretty, map[string]interface{}{
			"privatekey": map[string]interface{}{
				"hint": priv.Hint(),
				"key":  priv.String(),
			},
			"publickey": map[string]interface{}{
				"hint": priv.Publickey().Hint(),
				"key":  priv.Publickey().String(),
			},
		})
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "      hint: %s\n", priv.Hint().Verbose())
		_, _ = fmt.Fprintf(os.Stdout, "privatekey: %s\n", priv.String())
		_, _ = fmt.Fprintf(os.Stdout, " publickey: %s\n", priv.Publickey().String())
	}

	return nil
}
