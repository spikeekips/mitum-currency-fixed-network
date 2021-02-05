package cmds

import (
	"context"
	"reflect"

	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/storage"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum-currency/digest"
)

var (
	ContextValueDigestDesign  config.ContextKey = "digest_design"
	ContextValueDigestStorage config.ContextKey = "digest_storage"
	ContextValueDigestNetwork config.ContextKey = "digest_network"
	ContextValueDigester      config.ContextKey = "digester"
	ContextValueCurrencyPool  config.ContextKey = "currency_pool"
)

func LoadDigestDesignContextValue(ctx context.Context, l *DigestDesign) error {
	return config.LoadFromContextValue(ctx, ContextValueDigestDesign, l)
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
	return config.LoadFromContextValue(ctx, ContextValueDigestStorage, l)
}

func LoadDigestNetworkContextValue(ctx context.Context, l **digest.HTTP2Server) error {
	return config.LoadFromContextValue(ctx, ContextValueDigestNetwork, l)
}

func LoadDigesterContextValue(ctx context.Context, l **digest.Digester) error {
	return config.LoadFromContextValue(ctx, ContextValueDigester, l)
}

func LoadCurrencyPoolContextValue(ctx context.Context, l **currency.CurrencyPool) error {
	return config.LoadFromContextValue(ctx, ContextValueCurrencyPool, l)
}
