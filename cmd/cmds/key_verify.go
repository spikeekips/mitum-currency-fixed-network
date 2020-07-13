package cmds

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/logging"
)

type VerifyKeyCommand struct {
	Key    string `arg:"" name:"key" help:"key" optional:""`
	Detail bool   `name:"detail" short:"d" help:"print details"`
}

func (cmd *VerifyKeyCommand) Run(log logging.Logger) error {
	var s string
	if len(cmd.Key) > 0 {
		s = cmd.Key
		log.Debug().Str("input", s).Msg("load from argument")
	} else {
		var b []byte
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			sc := bufio.NewScanner(os.Stdin)
			for sc.Scan() {
				b = append(b, sc.Bytes()...)
			}

			if err := sc.Err(); err != nil {
				return err
			}
		}

		s = string(b)
		log.Debug().Str("input", s).Msg("load from stdin")
	}

	s = strings.TrimSpace(s)

	var pk key.Key
	if k, err := key.DecodeKey(defaultJSONEnc, s); err != nil {
		return err
	} else if err := k.IsValid(nil); err != nil {
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

	if priv != nil {
		_, _ = fmt.Fprintf(os.Stdout, "privatekey hint: %s\n", priv.Hint().Verbose())
		_, _ = fmt.Fprintf(os.Stdout, "     privatekey: %s\n", priv.String())
	}

	_, _ = fmt.Fprintf(os.Stdout, " publickey hint: %s\n", pub.Hint().Verbose())
	_, _ = fmt.Fprintf(os.Stdout, "      publickey: %s\n", pub.String())

	return nil
}
