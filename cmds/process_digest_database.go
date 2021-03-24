package cmds

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"

	"github.com/spikeekips/mitum-currency/digest"
)

const ProcessNameDigestDatabase = "digest_database"

var ProcessorDigestDatabase pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameDigestDatabase,
		[]string{process.ProcessNameNetwork},
		ProcessDigestDatabase,
	); err != nil {
		panic(err)
	} else {
		ProcessorDigestDatabase = i
	}
}

func ProcessDigestDatabase(ctx context.Context) (context.Context, error) {
	var design DigestDesign
	if err := LoadDigestDesignContextValue(ctx, &design); err != nil {
		if xerrors.Is(err, util.ContextValueNotFoundError) {
			return ctx, nil
		}

		return nil, err
	}

	var mst *mongodbstorage.Database
	if err := LoadDatabaseContextValue(ctx, &mst); err != nil {
		return ctx, err
	}

	if st, err := loadDigestDatabase(mst, false); err != nil {
		return ctx, err
	} else {
		var log logging.Logger
		if err := config.LoadLogContextValue(ctx, &log); err != nil {
			return ctx, err
		}

		_ = st.SetLogger(log)

		return context.WithValue(ctx, ContextValueDigestDatabase, st), nil
	}
}

func loadDigestDatabase(st *mongodbstorage.Database, readonly bool) (*digest.Database, error) {
	var mst, ost *mongodbstorage.Database
	if nst, err := st.New(); err != nil {
		return nil, err
	} else {
		mst = st
		ost = nst
	}

	var dst *digest.Database
	if readonly {
		if s, err := digest.NewReadonlyDatabase(mst, ost); err != nil {
			return nil, err
		} else {
			dst = s
		}
	} else {
		if s, err := digest.NewDatabase(mst, ost); err != nil {
			return nil, err
		} else {
			dst = s
		}
	}

	if err := dst.Initialize(); err != nil {
		return nil, err
	} else {
		return dst, nil
	}
}
