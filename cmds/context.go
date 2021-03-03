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
	ContextValueDigestDesign  util.ContextKey = "digest_design"
	ContextValueDigestStorage util.ContextKey = "digest_storage"
	ContextValueDigestNetwork util.ContextKey = "digest_network"
	ContextValueDigester      util.ContextKey = "digester"
	ContextValueCurrencyPool  util.ContextKey = "currency_pool"
)

func LoadDigestDesignContextValue(ctx context.Context, l *DigestDesign) error {
	return util.LoadFromContextValue(ctx, ContextValueDigestDesign, l)
}

func LoadStorageContextValue(ctx context.Context, l **mongodbstorage.Storage) error {
	st := (storage.Storage)(nil)
	if err := process.LoadStorageContextValue(ctx, &st); err != nil {
		return err
	}

	value := reflect.ValueOf(l)
	value.Elem().Set(reflect.ValueOf(st))

	return nil
}

func LoadDigestStorageContextValue(ctx context.Context, l **digest.Storage) error {
	return util.LoadFromContextValue(ctx, ContextValueDigestStorage, l)
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
