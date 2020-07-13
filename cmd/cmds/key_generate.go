package cmds

import (
	"fmt"
	"os"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/key"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type GenerateKeyCommand struct {
	Type   string `name:"type" help:"key type {btc ether stellar} (default: btc)" optional:"" default:"btc"`
	Format string `name:"format" help:"output format {normal json} (default: normal)" optional:"" default:"normal"`
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

	if len(cmd.Format) < 1 {
		cmd.Format = "normal"
	} else {
		switch cmd.Format {
		case "normal", "json":
		default:
			return xerrors.Errorf("unknown format: %q", cmd.Format)
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

	switch cmd.Format {
	case "json":
		_, _ = fmt.Fprintln(os.Stdout, string(jsonenc.MustMarshalIndent(map[string]interface{}{
			"type":       priv.Hint(),
			"privatekey": priv,
			"publickey":  priv.Publickey(),
		})))
	default:
		_, _ = fmt.Fprintf(os.Stdout, "      hint: %s\n", priv.Hint().Verbose())
		_, _ = fmt.Fprintf(os.Stdout, "privatekey: %s\n", priv.String())
		_, _ = fmt.Fprintf(os.Stdout, " publickey: %s\n", priv.Publickey().String())
	}

	return nil
}
