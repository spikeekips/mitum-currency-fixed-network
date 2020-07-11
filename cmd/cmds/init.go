package cmds

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"

	mc "github.com/spikeekips/mitum-currency"
)

type InitCommand struct {
	Design  string `arg:"" name:"node design file" help:"node design file" type:"existingfile"`
	Force   bool   `help:"clean the existing environment"`
	version util.Version
}

func (cmd *InitCommand) Run(flags *MainFlags, version util.Version) error {
	var log logging.Logger
	if l, err := setupLogging(flags.LogFlags); err != nil {
		return err
	} else {
		log = l
	}

	log.Info().Str("version", version.String()).Msg("mitum-currency")
	log.Debug().Interface("flags", flags).Msg("flags parsed")
	defer log.Info().Msg("mitum-currency finished")

	log.Info().Msg("trying to initialize")

	cmd.version = version

	return cmd.run(log)
}

func (cmd *InitCommand) run(log logging.Logger) error {
	var nr *mc.Launcher
	if n, err := createLauncherFromDesign(cmd.Design, cmd.version, log); err != nil {
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
	}

	log.Debug().Msg("checking existing blocks")

	if err := cmd.checkExisting(nr, log); err != nil {
		return err
	}

	log.Debug().Msg("trying to create genesis block")
	var ops []operation.Operation
	if o, err := cmd.loadInitOperations(nr); err != nil {
		return err
	} else {
		ops = o
	}

	log.Debug().Int("operations", len(ops)).Msg("operations loaded")

	if gg, err := isaac.NewGenesisBlockV0Generator(nr.Localstate(), ops); err != nil {
		return xerrors.Errorf("failed to create genesis block generator: %w", err)
	} else if blk, err := gg.Generate(); err != nil {
		return xerrors.Errorf("failed to generate genesis block: %w", err)
	} else {
		log.Info().
			Dict("block", logging.Dict().Hinted("height", blk.Height()).Hinted("hash", blk.Hash())).
			Msg("genesis block created")
	}

	log.Info().Msg("genesis block created")
	log.Info().Msg("iniialized")

	return nil
}

func (cmd *InitCommand) checkExisting(nr *mc.Launcher, log logging.Logger) error {
	log.Debug().Msg("checking existing blocks")

	var manifest block.Manifest
	if m, found, err := nr.Storage().LastManifest(); err != nil {
		return err
	} else if found {
		manifest = m
	}

	if manifest == nil {
		log.Debug().Msg("not found existing blocks")
	} else {
		log.Debug().Msgf("found existing blocks: block=%d", manifest.Height())

		log.Debug().Msg("existing environment cleaned")
	}

	return nil
}

func (cmd *InitCommand) loadInitOperations(nr *mc.Launcher) ([]operation.Operation, error) {
	var ops []operation.Operation
	if o, err := mc.LoadPolicyOperation(nr.Design()); err != nil {
		return nil, err
	} else {
		ops = append(ops, o...)
	}

	if o, err := mc.LoadOtherInitOperations(nr); err != nil {
		return nil, err
	} else {
		ops = append(ops, o...)
	}

	return ops, nil
}
