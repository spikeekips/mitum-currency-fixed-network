package cmds

import (
	"fmt"
	"os"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/logging"
)

type VerifyKeyCommand struct {
	Key    string `arg:"" name:"key" help:"key" optional:""`
	Detail bool   `name:"detail" short:"d" help:"print details"`
}

func (cmd *VerifyKeyCommand) Run(log logging.Logger) error {
	var pk key.Key
	if k, fromString, err := loadKeyFromFileOrInput(cmd.Key); err != nil {
		return err
	} else {
		pk = k
		log.Debug().Bool("from_string", fromString).Interface("key", pk).Msg("key parsed")
	}

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

	if priv != nil {
		_, _ = fmt.Fprintf(os.Stdout, "privatekey hint: %s\n", priv.Hint().Verbose())
		_, _ = fmt.Fprintf(os.Stdout, "     privatekey: %s\n", priv.String())
	}

	_, _ = fmt.Fprintf(os.Stdout, " publickey hint: %s\n", pub.Hint().Verbose())
	_, _ = fmt.Fprintf(os.Stdout, "      publickey: %s\n", pub.String())

	return nil
}
