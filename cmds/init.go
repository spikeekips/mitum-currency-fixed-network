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
	Design  FileLoad `arg:"" name:"node design file" help:"node design file"`
	Force   bool     `help:"clean the existing environment"`
	version util.Version
}

func (cmd *InitCommand) Run(flags *MainFlags, version util.Version) error {
	var log logging.Logger
	if l, err := SetupLogging(flags.Log, flags.LogFlags); err != nil {
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
	var nr *currency.Launcher
	if n, err := createLauncherFromDesign(cmd.Design.Bytes(), cmd.version, log); err != nil {
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

	log.Debug().Msg("checking existing blocks")

	if err := cmd.checkExisting(nr, log); err != nil {
		return err
	}

	var ops []operation.Operation
	if o, err := cmd.loadInitOperations(nr); err != nil {
		return err
	} else {
		ops = o
	}

	log.Debug().Int("operations", len(ops)).Msg("operations loaded")

	log.Debug().Msg("trying to create genesis block")
	var genesisBlock block.Block
	if gg, err := isaac.NewGenesisBlockV0Generator(nr.Localstate(), ops); err != nil {
		return xerrors.Errorf("failed to create genesis block generator: %w", err)
	} else if blk, err := gg.Generate(); err != nil {
		return xerrors.Errorf("failed to generate genesis block: %w", err)
	} else {
		log.Info().
			Dict("block", logging.Dict().Hinted("height", blk.Height()).Hinted("hash", blk.Hash())).
			Msg("genesis block created")

		genesisBlock = blk
	}

	log.Info().Msg("genesis block created")
	log.Info().Msg("iniialized")

	return cmd.saveGenesisAccount(nr, genesisBlock)
}

func (cmd *InitCommand) checkExisting(nr *currency.Launcher, log logging.Logger) error {
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

		return xerrors.Errorf("already blocks exist, clean first")
	}

	return nil
}

func (cmd *InitCommand) loadInitOperations(nr *currency.Launcher) ([]operation.Operation, error) {
	var ops []operation.Operation
	if o, err := currency.LoadPolicyOperation(nr.Design()); err != nil {
		return nil, err
	} else {
		ops = append(ops, o...)
	}

	if o, err := currency.LoadOtherInitOperations(nr); err != nil {
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

func (cmd *InitCommand) saveGenesisAccount(nr *currency.Launcher, genesisBlock block.Block) error {
	var gac currency.Account
	for i := range genesisBlock.States() {
		st := genesisBlock.States()[i]
		if currency.IsStateAccountKey(st.Key()) {
			if ac, err := currency.LoadStateAccountValue(st); err != nil {
				return err
			} else {
				gac = ac
			}
			break
		}
	}

	if gac.IsEmpty() {
		return xerrors.Errorf("failed to find genesis account")
	}

	return saveGenesisAccountInfo(nr.Storage(), gac)
}

func (cmd *InitCommand) prepareProposalProcessor(nr *currency.Launcher) error {
	return initlaizeProposalProcessor(
		// NOTE NilFeeAmount will be applied whatever design defined
		nr.ProposalProcessor(),
		currency.NewOperationProcessor(currency.NewNilFeeAmount(), nil),
	)
}
