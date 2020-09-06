package cmds

type SealCommand struct {
	Send          SendCommand          `cmd:"" name:"send" help:"send seal to remote mitum node"`
	CreateAccount CreateAccountCommand `cmd:"" name:"create-account" help:"create new account"`
	Transfer      TransferCommand      `cmd:"" name:"transfer" help:"transfer amount"`
	KeyUpdater    KeyUpdaterCommand    `cmd:"" name:"key-updater" help:"update keys"`
	Sign          SignSealCommand      `cmd:"" name:"sign" help:"sign seal"`
	SignFact      SignFactCommand      `cmd:"" name:"sign-fact" help:"sign facts of operation seal"`
}
