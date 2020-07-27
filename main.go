package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"

	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"

	"github.com/spikeekips/mitum-currency/cmds"
)

var Version string = "v0.0.1"

var mainHelpOptions = kong.HelpOptions{
	NoAppSummary: false,
	Compact:      false,
	Summary:      true,
	Tree:         false,
}

var mainDefaultVars = kong.Vars{
	"log":                      "",
	"log_level":                "error",
	"log_format":               "terminal",
	"log_color":                "false",
	"verbose":                  "false",
	"enable_pprofiling":        "false",
	"mem_prof_file":            "/tmp/mitum-currency-mem.prof",
	"trace_prof_file":          "/tmp/mitum-currency-trace.prof",
	"cpu_prof_file":            "/tmp/mitum-currency-cpu.prof",
	"node_url":                 "quic://localhost:54321",
	"create_account_threshold": "100",
}

func main() {
	flags := &cmds.MainFlags{
		LogFlags: &contestlib.LogFlags{},
		Run:      cmds.RunCommand{PprofFlags: &launcher.PprofFlags{}},
	}
	ctx := kong.Parse(
		flags,
		kong.Name(os.Args[0]),
		kong.Description("mitum currency"),
		kong.UsageOnError(),
		kong.ConfigureHelp(mainHelpOptions),
		mainDefaultVars,
	)

	version := util.Version(Version)
	ctx.FatalIfErrorf(func() error {
		return version.IsValid(nil)
	}())

	if ctx.Command() == "version" {
		_, _ = fmt.Fprintln(os.Stdout, Version)

		os.Exit(0)
	}

	ctx.FatalIfErrorf(run(flags, ctx, version))

	os.Exit(0)
}

func run(flags *cmds.MainFlags, ctx *kong.Context, version util.Version) error {
	defer contestlib.ExitHooks.Run()

	var log logging.Logger
	if l, err := cmds.SetupLogging(flags.Log, flags.LogFlags); err != nil {
		return err
	} else {
		log = l
	}

	log.Info().Str("version", version.String()).Msg("mitum-currency")
	log.Debug().Interface("flags", flags).Msg("flags parsed")
	defer log.Info().Msg("mitum-currency finished")

	return ctx.Run(flags, version, log)
}
