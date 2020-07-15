package cmds

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/automaxprocs/maxprocs"
	"golang.org/x/xerrors"

	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"

	mc "github.com/spikeekips/mitum-currency"
)

type RunCommand struct {
	*launcher.PprofFlags
	Design    FileLoad      `arg:"" name:"node design file" help:"node design file"`
	ExitAfter time.Duration `help:"exit after the given duration (default: none)" default:"0s"`
}

func (cmd *RunCommand) Run(version util.Version, log logging.Logger) error {
	log.Info().Msg("mitum-currency node started")

	_, _ = maxprocs.Set(maxprocs.Logger(func(f string, s ...interface{}) {
		log.Debug().Msgf(f, s...)
	}))

	if cancel, err := launcher.RunPprof(cmd.PprofFlags); err != nil {
		return err
	} else {
		contestlib.ExitHooks.Add(func() {
			if err := cancel(); err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err.Error())
			}
		})
	}

	var nr *mc.Launcher
	if n, err := createLauncherFromDesign(cmd.Design, version, log); err != nil {
		return xerrors.Errorf("failed to create node runner: %w", err)
	} else {
		nr = n
	}

	if err := nr.Initialize(); err != nil {
		return xerrors.Errorf("failed to generate node from design: %w", err)
	}

	if err := nr.Start(); err != nil {
		return xerrors.Errorf("failed to start: %w", err)
	}

	select {
	case err := <-nr.ErrChan():
		return err
	case <-func(w time.Duration) <-chan time.Time {
		if w < 1 {
			ch := make(chan time.Time)
			return ch
		}

		return time.After(w)
	}(cmd.ExitAfter):
		log.Info().Str("exit-after", cmd.ExitAfter.String()).Msg("expired, exit.")

		return nil
	}
}
