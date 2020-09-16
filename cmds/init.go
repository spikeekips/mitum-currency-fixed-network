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

	nr *Launcher
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
	if err := cmd.initialize(); err != nil {
		return err
	}

	cmd.Log().Debug().Msg("checking existing blocks")

	if err := cmd.checkExisting(); err != nil {
		return err
	}

	var ops []operation.Operation
	if o, err := cmd.loadInitOperations(); err != nil {
		return err
	} else {
		ops = o
	}

	cmd.Log().Debug().Int("operations", len(ops)).Msg("operations loaded")

	cmd.Log().Debug().Msg("trying to create genesis block")
	var genesisBlock block.Block
	if gg, err := isaac.NewGenesisBlockV0Generator(cmd.nr.Localstate(), ops); err != nil {
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

	if _, _, err := saveGenesisAccountInfo(cmd.nr.Storage(), genesisBlock, cmd.Log()); err != nil {
		return err
	}

	return nil
}

func (cmd *InitCommand) initialize() error {
	if n, err := createLauncherFromDesign(cmd.Design.Bytes(), cmd.version, cmd.Log()); err != nil {
		return err
	} else {
		cmd.nr = n
	}

	if err := cmd.nr.AttachStorage(); err != nil {
		return xerrors.Errorf("failed to attach storage: %w", err)
	}

	if cmd.Force {
		if err := cmd.nr.Storage().Clean(); err != nil {
			return xerrors.Errorf("failed to clean storage: %w", err)
		} else if err := cmd.nr.Localstate().BlockFS().Clean(false); err != nil {
			return xerrors.Errorf("failed to clean blockfs: %w", err)
		}
	}

	if err := cmd.nr.Initialize(); err != nil {
		return xerrors.Errorf("failed to generate node from design: %w", err)
	} else if err := cmd.prepareProposalProcessor(); err != nil {
		return err
	}

	return nil
}

func (cmd *InitCommand) checkExisting() error {
	cmd.Log().Debug().Msg("checking existing blocks")

	var manifest block.Manifest
	if m, found, err := cmd.nr.Storage().LastManifest(); err != nil {
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

func (cmd *InitCommand) loadInitOperations() ([]operation.Operation, error) {
	var ops []operation.Operation
	if o, err := LoadPolicyOperation(cmd.nr.Design()); err != nil {
		return nil, err
	} else {
		ops = append(ops, o...)
	}

	if o, err := LoadOtherInitOperations(cmd.nr); err != nil {
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

func (cmd *InitCommand) prepareProposalProcessor() error {
	return initlaizeProposalProcessor(
		// NOTE NilFeeAmount will be applied whatever design defined
		cmd.nr.ProposalProcessor(),
		currency.NewOperationProcessor(currency.NewNilFeeAmount(), nil),
	)
}
