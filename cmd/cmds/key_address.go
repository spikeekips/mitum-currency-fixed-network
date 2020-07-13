package cmds

import (
	"bytes"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"

	mc "github.com/spikeekips/mitum-currency"
)

type KeyAddressCommand struct {
	Keys []KeyFlag `arg:"" name:"key" help:"key for address (ex: \"<public key>,<weight>\")" sep:"@" optional:""`
}

func (cmd *KeyAddressCommand) Run() error {
	if b, err := loadFromStdInput(); err != nil {
		return err
	} else if len(b) > 0 {
		kf := new(KeyFlag)
		if err := kf.UnmarshalText(bytes.TrimSpace(b)); err != nil {
			return err
		}

		log.Debug().Str("input", string(b)).Interface("key", kf.Key).Msg("load from stdin")

		cmd.Keys = append(cmd.Keys, *kf)
	}

	log.Debug().Interface("keys", cmd.Keys).Msg("keys loaded")

	keys := make([]mc.Key, len(cmd.Keys))
	for i := range cmd.Keys {
		keys[i] = cmd.Keys[i].Key
	}

	if a, err := mc.NewAddressFromKeys(keys); err != nil {
		return err
	} else {
		_, _ = fmt.Fprintln(os.Stdout, a.String())
	}

	return nil
}
