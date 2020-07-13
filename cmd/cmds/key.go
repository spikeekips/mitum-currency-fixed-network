package cmds

const (
	btc     = "btc"
	ether   = "ether"
	stellar = "stellar"
)

type KeyCommand struct {
	Generate GenerateKeyCommand `cmd:"" help:"generate keypair"`
	Verify   VerifyKeyCommand   `cmd:"" help:"verify key"`
	Address  KeyAddressCommand  `cmd:"" help:"generate address from key"`
}
