package cmds

import (
	"encoding/base64"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
)

type SignKeyCommand struct {
	*BaseCommand
	Key   StringLoad `arg:"" name:"privatekey" help:"privatekey" required:"true"`
	Base  string     `arg:"" name:"signature base" help:"signature base for signing" required:"true"`
	Quite bool       `name:"quite" short:"q" help:"keep silence"`
}

func NewSignKeyCommand() SignKeyCommand {
	return SignKeyCommand{
		BaseCommand: NewBaseCommand("key-sign"),
	}
}

func (cmd *SignKeyCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return errors.Wrap(err, "failed to initialize command")
	}

	var priv key.Privatekey
	if k, err := loadKey(cmd.Key.Bytes()); err != nil {
		if cmd.Quite {
			os.Exit(1) // revive:disable-line:deep-exit
		}

		return err
	} else if pk, ok := k.(key.Privatekey); !ok {
		return errors.Errorf("not Privatekey, %T", k)
	} else {
		priv = pk
	}

	cmd.Log().Debug().Interface("key", priv).Msg("key parsed")

	var base []byte
	if s := strings.TrimSpace(cmd.Base); len(s) < 1 {
		return errors.Errorf("empty signature base")
	} else if b, err := base64.StdEncoding.DecodeString(s); err != nil {
		return errors.Wrap(err, "invalid signature base; failed to decode by base64")
	} else {
		base = b
	}

	sig, err := priv.Sign(base)
	if err != nil {
		return errors.Wrap(err, "failed to sign base")
	}
	cmd.print(sig.String())

	return nil
}

func loadKey(b []byte) (key.Key, error) {
	s := strings.TrimSpace(string(b))

	if pk, err := key.DecodeKey(jenc, s); err != nil {
		return nil, err
	} else if err := pk.IsValid(nil); err != nil {
		return nil, err
	} else {
		return pk, nil
	}
}
