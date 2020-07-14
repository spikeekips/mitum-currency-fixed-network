package cmds

import (
	"fmt"
	"os"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/logging"
)

type VerifyKeyCommand struct {
	Key    StringLoad `arg:"" name:"key" help:"key" optional:""`
	Detail bool       `name:"detail" short:"d" help:"print details"`
	JSON   bool       `name:"json" help:"json output format (default: false)" optional:"" default:"false"`
	Pretty bool       `name:"pretty" help:"pretty format"`
}

func (cmd *VerifyKeyCommand) Run(log logging.Logger) error {
	var pk key.Key
	if k, err := loadKey(cmd.Key.Bytes()); err != nil {
		return err
	} else {
		pk = k
	}

	log.Debug().Interface("key", pk).Msg("key parsed")

	if !cmd.Detail {
		return nil
	}

	var priv key.Privatekey
	var pub key.Publickey
	switch t := pk.(type) {
	case key.Privatekey:
		priv = t
		pub = t.Publickey()
	case key.Publickey:
		pub = t
	}

	if cmd.JSON {
		m := map[string]interface{}{
			"publickey": map[string]interface{}{
				"hint": pub.Hint(),
				"key":  pub.String(),
			},
		}

		if priv != nil {
			m["privtekey"] = map[string]interface{}{
				"hint": priv.Hint(),
				"key":  priv.String(),
			}
		}

		prettyPrint(cmd.Pretty, m)

		return nil
	}

	if priv != nil {
		_, _ = fmt.Fprintf(os.Stdout, "privatekey hint: %s\n", priv.Hint().Verbose())
		_, _ = fmt.Fprintf(os.Stdout, "     privatekey: %s\n", priv.String())
	}

	_, _ = fmt.Fprintf(os.Stdout, " publickey hint: %s\n", pub.Hint().Verbose())
	_, _ = fmt.Fprintf(os.Stdout, "      publickey: %s\n", pub.String())

	return nil
}
