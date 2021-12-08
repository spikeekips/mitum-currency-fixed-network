package cmds

import (
	"fmt"

	"github.com/spikeekips/mitum/base/key"
)

const (
	btc     = "btc"
	ether   = "ether"
	stellar = "stellar"
)

type KeyCommand struct {
	New     GenerateKeyCommand `cmd:"" help:"new keypair"`
	Verify  VerifyKeyCommand   `cmd:"" help:"verify key"`
	Address KeyAddressCommand  `cmd:"" help:"generate address from key"`
	Sign    SignKeyCommand     `cmd:"" help:"signature signing"`
}

func NewKeyCommand() KeyCommand {
	return KeyCommand{
		New:     NewGenerateKeyCommand(),
		Verify:  NewVerifyKeyCommand(),
		Address: NewKeyAddressCommand(),
		Sign:    NewSignKeyCommand(),
	}
}

func IsValidKeyType(s string) bool {
	switch s {
	case btc, ether, stellar:
		return true
	default:
		return false
	}
}

func GenerateKey(seed string) (key.Privatekey, error) {
	switch l := len(seed); {
	case l < 1:
		return key.NewBasePrivatekey(), nil
	case l < key.MinSeedSize:
		return nil, fmt.Errorf("seed should be over %d < %d", l, key.MinSeedSize)
	default:
		return key.NewBasePrivatekeyFromSeed(seed)
	}
}
