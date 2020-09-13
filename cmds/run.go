package cmds

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/automaxprocs/maxprocs"
	"golang.org/x/xerrors"

	"github.com/rs/zerolog/log"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"

	"github.com/spikeekips/mitum-currency/currency"
)

type RunCommand struct {
	*launcher.PprofFlags
	Design    FileLoad      `arg:"" name:"node design file" help:"node design file"`
	ExitAfter time.Duration `help:"exit after the given duration (default: none)" default:"0s"`
	nr        *Launcher
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

	if n, err := createLauncherFromDesign(cmd.Design, version, log); err != nil {
		return xerrors.Errorf("failed to create node runner: %w", err)
	} else {
		cmd.nr = n
	}

	if err := cmd.initialize(); err != nil {
		return err
	}

	contestlib.ConnectSignal()
	defer contestlib.ExitHooks.Run()

	if err := cmd.nr.Start(); err != nil {
		return xerrors.Errorf("failed to start: %w", err)
	}

	select {
	case err := <-cmd.nr.ErrChan():
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

func (cmd *RunCommand) prepareProposalProcessor() error {
	var fa currency.FeeAmount
	var feeReceiverFunc func() (base.Address, error)
	if cmd.nr.Design().FeeAmount == nil {
		fa = currency.NewNilFeeAmount()

		log.Debug().Msg("fee not applied")
	} else {
		fa = cmd.nr.Design().FeeAmount

		var to base.Address
		if cmd.nr.Design().FeeReceiver != nil {
			to = cmd.nr.Design().FeeReceiver

			switch _, found, err := cmd.nr.Storage().State(currency.StateKeyAccount(to)); {
			case err != nil:
				return xerrors.Errorf("failed to find fee receiver, %v: %w", to, err)
			case !found:
				return xerrors.Errorf("fee receiver, %v does not exist", to)
			}
		} else if gac, _, exists := cmd.nr.genesisInfo(); exists {
			to = gac.Address()
		}

		if to != nil {
			feeReceiverFunc = func() (base.Address, error) {
				return to, nil
			}
		} else {
			feeReceiverFunc = func() (base.Address, error) {
				if gac, _, exists := cmd.nr.genesisInfo(); exists {
					return gac.Address(), nil
				} else {
					return nil, nil
				}
			}
		}

		log.Debug().Str("fee_amount", cmd.nr.Design().FeeAmount.Verbose()).Interface("fee_receiver", to).Msg("fee applied")
	}

	return initlaizeProposalProcessor(
		cmd.nr.ProposalProcessor(),
		currency.NewOperationProcessor(fa, feeReceiverFunc),
	)
}

func (cmd *RunCommand) whenBlockSaved(blocks []block.Block) {
	if _, _, exists := cmd.nr.genesisInfo(); exists {
		return
	}

	// NOTE catch genesis block
	var genesisBlock block.Block
	for _, blk := range blocks {
		if blk.Height() == base.Height(0) {
			genesisBlock = blk

			break
		}
	}
	if genesisBlock == nil {
		return
	}

	log.Debug().Msg("trying to find genesis block")
	if ga, gb, err := saveGenesisAccountInfo(cmd.nr.Storage(), genesisBlock); err != nil {
		log.Error().Err(err).Msg("failed to save genesis account to node info")

		cmd.nr.setGenesisInfo(currency.Account{}, currency.ZeroAmount)
	} else {
		cmd.nr.setGenesisInfo(ga, gb)
	}
}

func (cmd *RunCommand) initialize() error {
	if err := cmd.nr.Initialize(); err != nil {
		return xerrors.Errorf("failed to generate node from design: %w", err)
	} else if err := cmd.prepareProposalProcessor(); err != nil {
		return err
	}

	if cs := cmd.nr.ConsensusStates().StateHandler(base.StateConsensus); cs != nil {
		cs.(*isaac.StateConsensusHandler).WhenBlockSaved(cmd.whenBlockSaved)
	}
	if cs := cmd.nr.ConsensusStates().StateHandler(base.StateSyncing); cs != nil {
		cs.(*isaac.StateSyncingHandler).WhenBlockSaved(cmd.whenBlockSaved)
	}

	if ac, ba, err := loadGenesisAccountInfo(cmd.nr.Storage()); err != nil {
		log.Error().Err(err).Msg("failed to load genesis account info")
	} else {
		cmd.nr.setGenesisInfo(ac, ba) // NOTE set for NodeInfo
	}

	return nil
}
