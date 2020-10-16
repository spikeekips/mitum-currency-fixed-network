package cmds

import (
	"encoding/base64"
	"os"
	"strings"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type SignKeyCommand struct {
	BaseCommand
	Key   StringLoad `arg:"" name:"privatekey" help:"privatekey" required:""`
	Base  string     `arg:"" name:"signature base" help:"signature base for signing" required:""`
	Quite bool       `name:"quite" short:"q" help:"keep silence"`
}

func (cmd *SignKeyCommand) Run(flags *MainFlags, version util.Version, log logging.Logger) error {
	_ = cmd.BaseCommand.Run(flags, version, log)

	var priv key.Privatekey
	if k, err := loadKey(cmd.Key.Bytes()); err != nil {
		if cmd.Quite {
			os.Exit(1)
		}

		return err
	} else if pk, ok := k.(key.Privatekey); !ok {
		return xerrors.Errorf("not Privatekey, %T", k)
	} else {
		priv = pk
	}

	cmd.Log().Debug().Interface("key", priv).Msg("key parsed")

	var base []byte
	if s := strings.TrimSpace(cmd.Base); len(s) < 1 {
		return xerrors.Errorf("empty signature base")
	} else if b, err := base64.StdEncoding.DecodeString(s); err != nil {
		return xerrors.Errorf("invalid signature base; failed to decode by base64: %w", err)
	} else {
		base = b
	}

	if sig, err := priv.Sign(base); err != nil {
		return xerrors.Errorf("failed to sign base: %w", err)
	} else {
		cmd.print(sig.String())
	}

	return nil
}
