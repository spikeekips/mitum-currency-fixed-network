package cmds

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/automaxprocs/maxprocs"
	"golang.org/x/xerrors"

	"github.com/rs/zerolog/log"
	"github.com/spikeekips/mitum/base"
	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"

	"github.com/spikeekips/mitum-currency/currency"
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

	var nr *currency.Launcher
	if n, err := createLauncherFromDesign(cmd.Design, version, log); err != nil {
		return xerrors.Errorf("failed to create node runner: %w", err)
	} else {
		nr = n
	}

	if err := nr.Initialize(); err != nil {
		return xerrors.Errorf("failed to generate node from design: %w", err)
	} else if err := cmd.prepareProposalProcessor(nr); err != nil {
		return err
	}

	contestlib.ConnectSignal()
	defer contestlib.ExitHooks.Run()

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

func (cmd *RunCommand) prepareProposalProcessor(nr *currency.Launcher) error {
	var fa currency.FeeAmount
	var feeReceiverFunc func() (base.Address, error)
	if nr.Design().FeeAmount == nil {
		fa = currency.NewNilFeeAmount()

		log.Debug().Msg("fee not applied")
	} else {
		fa = nr.Design().FeeAmount

		var to base.Address
		if nr.Design().FeeReceiver != nil {
			to = nr.Design().FeeReceiver

			switch _, found, err := nr.Storage().State(currency.StateKeyAccount(to)); {
			case err != nil:
				return xerrors.Errorf("failed to find fee receiver, %v: %w", to, err)
			case !found:
				return xerrors.Errorf("fee receiver, %v does not exist", to)
			}
		} else {
			if ac, err := loadGenesisAccountInfo(nr.Storage()); err != nil {
				return err
			} else {
				to = ac.Address()
			}
		}

		if to == nil {
			return xerrors.Errorf("fee receiver not found")
		}

		feeReceiverFunc = func() (base.Address, error) {
			return to, nil
		}

		log.Debug().Str("fee_amount", nr.Design().FeeAmount.Verbose()).Str("fee_receiver", to.String()).Msg("fee applied")
	}

	return initlaizeProposalProcessor(
		nr.ProposalProcessor(),
		currency.NewOperationProcessor(fa, feeReceiverFunc),
	)
}
