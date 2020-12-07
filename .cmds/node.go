package cmds

type NodeCommand struct {
	NodeInfo NodeInfoCommand `cmd:"" name:"info" help:"get node info from mitum node"`
}
