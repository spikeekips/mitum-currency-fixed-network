package cmds

import (
	"context"

	"github.com/spikeekips/mitum-currency/digest"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/storage"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

const ProcessNameDigestStorage = "digest_storage"

var ProcessorDigestStorage pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameDigestStorage,
		[]string{process.ProcessNameNetwork},
		ProcessDigestStorage,
	); err != nil {
		panic(err)
	} else {
		ProcessorDigestStorage = i
	}
}

func ProcessDigestStorage(ctx context.Context) (context.Context, error) {
	var design DigestDesign
	if err := LoadDigestDesignContextValue(ctx, &design); err != nil {
		if xerrors.Is(err, config.ContextValueNotFoundError) {
			return ctx, nil
		}

		return nil, err
	}

	var mst storage.Storage
	if err := process.LoadStorageContextValue(ctx, &mst); err != nil {
		return ctx, err
	}

	if st, err := loadDigestStorage(mst, false); err != nil {
		return ctx, err
	} else {
		var log logging.Logger
		if err := config.LoadLogContextValue(ctx, &log); err != nil {
			return ctx, err
		}

		_ = st.SetLogger(log)

		return context.WithValue(ctx, ContextValueDigestStorage, st), nil
	}
}

func loadDigestStorage(st storage.Storage, readonly bool) (*digest.Storage, error) {
	var mst, ost *mongodbstorage.Storage
	if s, ok := st.(*mongodbstorage.Storage); !ok {
		return nil, xerrors.Errorf("digest needs *mongodbstorage.Storage, not %T", st)
	} else if rst, err := s.Readonly(); err != nil {
		return nil, err
	} else {
		mst = rst
		ost = s
	}

	var nst *digest.Storage
	if readonly {
		if s, err := digest.NewReadonlyStorage(mst, ost); err != nil {
			return nil, err
		} else {
			nst = s
		}
	} else {
		if s, err := digest.NewStorage(mst, ost); err != nil {
			return nil, err
		} else {
			nst = s
		}
	}

	if err := nst.Initialize(); err != nil {
		return nil, err
	} else {
		return nst, nil
	}
}
