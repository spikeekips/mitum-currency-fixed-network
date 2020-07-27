package cmds

const (
	btc     = "btc"
	ether   = "ether"
	stellar = "stellar"
)

type KeyCommand struct {
	New     GenerateKeyCommand `cmd:"" help:"new keypair"`
	Verify  VerifyKeyCommand   `cmd:"" help:"verify key"`
	Address KeyAddressCommand  `cmd:"" help:"generate address from key"`
}
