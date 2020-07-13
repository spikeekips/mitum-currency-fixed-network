package cmds

type SealCommand struct {
	CreateAccount CreateAccountCommand `cmd:"" name:"create-account" help:"create new account"`
	Transfer      TransferCommand      `cmd:"" name:"transfer" help:"transfer amount"`
	Sign          SignSealCommand      `cmd:"" name:"sign" help:"sign seal"`
	SignFact      SignFactCommand      `cmd:"" name:"sign-fact" help:"sign facts of operation seal"`
}
