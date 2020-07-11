package cmds

type SendCommand struct {
	CreateAccount CreateAccountCommand `cmd:"" name:"create-account" help:"create new account"`
}
