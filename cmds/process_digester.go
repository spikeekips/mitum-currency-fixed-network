package cmds

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum-currency/digest"
)

const (
	ProcessNameDigester      = "digester"
	ProcessNameStartDigester = "start_digester"
	HookNameDigesterFollowUp = "followup_digester"
)

var (
	ProcessorDigester      pm.Process
	ProcessorStartDigester pm.Process
)

func init() {
	if i, err := pm.NewProcess(
		ProcessNameDigester,
		[]string{ProcessNameDigestDatabase},
		ProcessDigester,
	); err != nil {
		panic(err)
	} else {
		ProcessorDigester = i
	}

	if i, err := pm.NewProcess(
		ProcessNameStartDigester,
		[]string{ProcessNameDigester},
		ProcessStartDigester,
	); err != nil {
		panic(err)
	} else {
		ProcessorStartDigester = i
	}
}

func ProcessDigester(ctx context.Context) (context.Context, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var st *digest.Database
	if err := LoadDigestDatabaseContextValue(ctx, &st); err != nil {
		if xerrors.Is(err, util.ContextValueNotFoundError) {
			return ctx, nil
		}

		return ctx, err
	}

	di := digest.NewDigester(st, nil)
	_ = di.SetLogger(log)

	return context.WithValue(ctx, ContextValueDigester, di), nil
}

func ProcessStartDigester(ctx context.Context) (context.Context, error) {
	var di *digest.Digester
	if err := LoadDigesterContextValue(ctx, &di); err != nil {
		if xerrors.Is(err, util.ContextValueNotFoundError) {
			return ctx, nil
		}

		return ctx, err
	}

	return ctx, di.Start()
}

func HookDigesterFollowUp(ctx context.Context) (context.Context, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	log.Debug().Msg("digester trying to follow up")

	var mst storage.Database
	if err := process.LoadDatabaseContextValue(ctx, &mst); err != nil {
		return ctx, err
	}

	var st *digest.Database
	if err := LoadDigestDatabaseContextValue(ctx, &st); err != nil {
		if xerrors.Is(err, util.ContextValueNotFoundError) {
			return ctx, nil
		}

		return ctx, err
	}

	switch m, found, err := mst.LastManifest(); {
	case err != nil:
		return ctx, err
	case !found:
		log.Debug().Msg("last manifest not found")
	case m.Height() > st.LastBlock():
		log.Info().
			Hinted("last_manifest", m.Height()).
			Hinted("last_block", st.LastBlock()).
			Msg("new blocks found to digest")

		if err := digestFollowup(ctx, m.Height()); err != nil {
			log.Error().Err(err).Msg("failed to follow up")

			return ctx, err
		} else {
			log.Info().Msg("digested new blocks")
		}
	default:
		log.Info().Msg("digested blocks is up-to-dated")
	}

	return ctx, nil
}

func digestFollowup(ctx context.Context, height base.Height) error {
	var st *digest.Database
	if err := LoadDigestDatabaseContextValue(ctx, &st); err != nil {
		return err
	}

	var blockData *localfs.BlockData
	{
		var bd blockdata.BlockData
		if err := process.LoadBlockDataContextValue(ctx, &bd); err != nil {
			return err
		} else if i, ok := bd.(*localfs.BlockData); !ok {
			return xerrors.Errorf("other block data, %T not yet supported", bd)
		} else {
			blockData = i
		}
	}

	var cp *currency.CurrencyPool
	if err := LoadCurrencyPoolContextValue(ctx, &cp); err != nil {
		return err
	}

	if height <= st.LastBlock() {
		return nil
	}

	lastBlock := st.LastBlock()
	if lastBlock < base.PreGenesisHeight {
		lastBlock = base.PreGenesisHeight
	}

	for i := lastBlock; i <= height; i++ {
		if blk, err := localfs.LoadBlock(blockData, i); err != nil {
			return err
		} else if err := digest.DigestBlock(st, blk); err != nil {
			return err
		}
	}

	return nil
}
