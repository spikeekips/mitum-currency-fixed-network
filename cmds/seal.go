package cmds

type SealCommand struct {
	Send                  SendCommand                  `cmd:"" name:"send" help:"send seal to remote mitum node"`
	CreateAccount         CreateAccountCommand         `cmd:"" name:"create-account" help:"create new account"`
	Transfer              TransferCommand              `cmd:"" name:"transfer" help:"transfer big"`
	KeyUpdater            KeyUpdaterCommand            `cmd:"" name:"key-updater" help:"update keys"`
	CurrencyRegister      CurrencyRegisterCommand      `cmd:"" name:"currency-register" help:"register new currency"`
	CurrencyPolicyUpdater CurrencyPolicyUpdaterCommand `cmd:"" name:"currency-policy-updater" help:"update currency policy"`  // revive:disable-line:line-length-limit
	SuffrageInflation     SuffrageInflationCommand     `cmd:"" name:"suffrage-inflation" help:"suffrage inflation operation"` // revive:disable-line:line-length-limit
	Sign                  SignSealCommand              `cmd:"" name:"sign" help:"sign seal"`
	SignFact              SignFactCommand              `cmd:"" name:"sign-fact" help:"sign facts of operation seal"`
}

func NewSealCommand() SealCommand {
	return SealCommand{
		Send:                  NewSendCommand(),
		CreateAccount:         NewCreateAccountCommand(),
		Transfer:              NewTransferCommand(),
		KeyUpdater:            NewKeyUpdaterCommand(),
		CurrencyRegister:      NewCurrencyRegisterCommand(),
		CurrencyPolicyUpdater: NewCurrencyPolicyUpdaterCommand(),
		SuffrageInflation:     NewSuffrageInflationCommand(),
		Sign:                  NewSignSealCommand(),
		SignFact:              NewSignFactCommand(),
	}
}
