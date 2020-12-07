package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/spikeekips/mitum-currency/cmds"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/util"
)

var (
	Version string = "v0.0.0"
	options        = []kong.Option{
		kong.Name("mitum-currency"),
		kong.Description("mitum-currency tool"),
		cmds.KeyAddressVars,
		cmds.SendVars,
	}
)

type mainflags struct {
	Node cmds.NodeCommand `cmd:"" help:"node"`
	// TODO Blocks mitumcmds.BlocksCommand `cmd:"" help:"get block data from node"`
	Key  cmds.KeyCommand  `cmd:"" help:"key"`
	Seal cmds.SealCommand `cmd:"" help:"seal"`
}

func main() {
	var nodeCommand cmds.NodeCommand
	if i, err := cmds.NewNodeCommand(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %+v\n", err)

		os.Exit(1)
	} else {
		nodeCommand = i
	}

	flags := mainflags{
		Node: nodeCommand,
		Key:  cmds.NewKeyCommand(),
		Seal: cmds.NewSealCommand(),
	}

	var kctx *kong.Context
	if i, err := mitumcmds.Context(os.Args[1:], &flags, options...); err != nil {
		fmt.Fprintf(os.Stderr, "error: %+v\n", err)

		os.Exit(1)
	} else {
		kctx = i
	}

	version := util.Version(Version)
	if err := version.IsValid(nil); err != nil {
		kctx.FatalIfErrorf(err)
	}

	if kctx.Command() == "version" {
		_, _ = fmt.Fprintln(os.Stdout, version)

		os.Exit(0)
	}

	if err := kctx.Run(version); err != nil {
		kctx.FatalIfErrorf(err)
	}

	os.Exit(0)
}
