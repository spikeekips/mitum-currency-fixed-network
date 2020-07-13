package cmds

type SealCommand struct {
	CreateAccount CreateAccountCommand `cmd:"" name:"create-account" help:"create new account"`
	SignFact      SignFactCommand      `cmd:"" name:"sign-fact" help:"sign facts of operation seal"`
}
