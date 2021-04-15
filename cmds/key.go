package cmds

import "github.com/spikeekips/mitum/base/key"

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

func GenerateKey(s string) key.Privatekey {
	switch s {
	case btc:
		return key.MustNewBTCPrivatekey()
	case ether:
		return key.MustNewEtherPrivatekey()
	case stellar:
		return key.MustNewStellarPrivatekey()
	default:
		return nil
	}
}
