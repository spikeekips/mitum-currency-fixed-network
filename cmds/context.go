package cmds

import (
	"context"

	"github.com/spikeekips/mitum-currency/digest"
	"github.com/spikeekips/mitum/launch/config"
)

var (
	ContextValueFeeDesign     config.ContextKey = "fee_design"
	ContextValueDigestDesign  config.ContextKey = "digest_design"
	ContextValueDigestStorage config.ContextKey = "digest_storage"
	ContextValueDigestNetwork config.ContextKey = "digest_network"
	ContextValueDigester      config.ContextKey = "digester"
)

func LoadFeeDesignContextValue(ctx context.Context, l *FeeDesign) error {
	return config.LoadFromContextValue(ctx, ContextValueFeeDesign, l)
}

func LoadDigestDesignContextValue(ctx context.Context, l *DigestDesign) error {
	return config.LoadFromContextValue(ctx, ContextValueDigestDesign, l)
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
