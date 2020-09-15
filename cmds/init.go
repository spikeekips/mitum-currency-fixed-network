package cmds

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"

	"github.com/spikeekips/mitum-currency/currency"
)

type InitCommand struct {
	BaseCommand
	Design FileLoad `arg:"" name:"node design file" help:"node design file"`
	Force  bool     `help:"clean the existing environment"`
}

func (cmd *InitCommand) Run(flags *MainFlags, version util.Version, log logging.Logger) error {
	_ = cmd.BaseCommand.Run(flags, version, log)

	if l, err := SetupLogging(flags.Log, flags.LogFlags); err != nil {
		return err
	} else {
		cmd.log = l
	}

	cmd.Log().Info().Str("version", version.String()).Msg("mitum-currency")
	cmd.Log().Debug().Interface("flags", flags).Msg("flags parsed")
	defer cmd.Log().Info().Msg("mitum-currency finished")

	cmd.Log().Info().Msg("trying to initialize")

	return cmd.run()
}

func (cmd *InitCommand) run() error {
	var nr *Launcher
	if n, err := createLauncherFromDesign(cmd.Design.Bytes(), cmd.version, cmd.Log()); err != nil {
		return err
	} else {
		nr = n
	}

	if err := nr.AttachStorage(); err != nil {
		return xerrors.Errorf("failed to attach storage: %w", err)
	}

	if cmd.Force {
		if err := nr.Storage().Clean(); err != nil {
			return xerrors.Errorf("failed to clean storage: %w", err)
		}
	}

	if err := nr.Initialize(); err != nil {
		return xerrors.Errorf("failed to generate node from design: %w", err)
	} else if err := cmd.prepareProposalProcessor(nr); err != nil {
		return err
	}

	cmd.Log().Debug().Msg("checking existing blocks")

	if err := cmd.checkExisting(nr); err != nil {
		return err
	}

	var ops []operation.Operation
	if o, err := cmd.loadInitOperations(nr); err != nil {
		return err
	} else {
		ops = o
	}

	cmd.Log().Debug().Int("operations", len(ops)).Msg("operations loaded")

	cmd.Log().Debug().Msg("trying to create genesis block")
	var genesisBlock block.Block
	if gg, err := isaac.NewGenesisBlockV0Generator(nr.Localstate(), ops); err != nil {
		return xerrors.Errorf("failed to create genesis block generator: %w", err)
	} else if blk, err := gg.Generate(); err != nil {
		return xerrors.Errorf("failed to generate genesis block: %w", err)
	} else {
		cmd.Log().Info().
			Dict("block", logging.Dict().Hinted("height", blk.Height()).Hinted("hash", blk.Hash())).
			Msg("genesis block created")

		genesisBlock = blk
	}

	cmd.Log().Info().Msg("genesis block created")
	cmd.Log().Info().Msg("iniialized")

	if _, _, err := saveGenesisAccountInfo(nr.Storage(), genesisBlock, cmd.Log()); err != nil {
		return err
	}

	return nil
}

func (cmd *InitCommand) checkExisting(nr *Launcher) error {
	cmd.Log().Debug().Msg("checking existing blocks")

	var manifest block.Manifest
	if m, found, err := nr.Storage().LastManifest(); err != nil {
		return err
	} else if found {
		manifest = m
	}

	if manifest == nil {
		cmd.Log().Debug().Msg("not found existing blocks")
	} else {
		cmd.Log().Debug().Msgf("found existing blocks: block=%d", manifest.Height())

		return xerrors.Errorf("already blocks exist, clean first")
	}

	return nil
}

func (cmd *InitCommand) loadInitOperations(nr *Launcher) ([]operation.Operation, error) {
	var ops []operation.Operation
	if o, err := LoadPolicyOperation(nr.Design()); err != nil {
		return nil, err
	} else {
		ops = append(ops, o...)
	}

	if o, err := LoadOtherInitOperations(nr); err != nil {
		return nil, err
	} else {
		ops = append(ops, o...)
	}

	var genesisOpFound bool
	for _, op := range ops {
		if _, ok := op.(currency.GenesisAccount); ok {
			genesisOpFound = true

			break
		}
	}

	if !genesisOpFound {
		return nil, xerrors.Errorf("GenesisAccount operation is missing")
	}

	return ops, nil
}

func (cmd *InitCommand) prepareProposalProcessor(nr *Launcher) error {
	return initlaizeProposalProcessor(
		// NOTE NilFeeAmount will be applied whatever design defined
		nr.ProposalProcessor(),
		currency.NewOperationProcessor(currency.NewNilFeeAmount(), nil),
	)
}
