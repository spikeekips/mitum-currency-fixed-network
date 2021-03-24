package cmds

import (
	"context"
	"reflect"

	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/storage"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum-currency/digest"
)

var (
	ContextValueDigestDesign   util.ContextKey = "digest_design"
	ContextValueDigestDatabase util.ContextKey = "digest_database"
	ContextValueDigestNetwork  util.ContextKey = "digest_network"
	ContextValueDigester       util.ContextKey = "digester"
	ContextValueCurrencyPool   util.ContextKey = "currency_pool"
)

func LoadDigestDesignContextValue(ctx context.Context, l *DigestDesign) error {
	return util.LoadFromContextValue(ctx, ContextValueDigestDesign, l)
}

func LoadDatabaseContextValue(ctx context.Context, l **mongodbstorage.Database) error {
	st := (storage.Database)(nil)
	if err := process.LoadDatabaseContextValue(ctx, &st); err != nil {
		return err
	}

	value := reflect.ValueOf(l)
	value.Elem().Set(reflect.ValueOf(st))

	return nil
}

func LoadDigestDatabaseContextValue(ctx context.Context, l **digest.Database) error {
	return util.LoadFromContextValue(ctx, ContextValueDigestDatabase, l)
}

func LoadDigestNetworkContextValue(ctx context.Context, l **digest.HTTP2Server) error {
	return util.LoadFromContextValue(ctx, ContextValueDigestNetwork, l)
}

func LoadDigesterContextValue(ctx context.Context, l **digest.Digester) error {
	return util.LoadFromContextValue(ctx, ContextValueDigester, l)
}

func LoadCurrencyPoolContextValue(ctx context.Context, l **currency.CurrencyPool) error {
	return util.LoadFromContextValue(ctx, ContextValueCurrencyPool, l)
}
