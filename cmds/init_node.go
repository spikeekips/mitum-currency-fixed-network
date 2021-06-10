package cmds

import (
	"context"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

var (
	GenesisAccountKey = "genesis_account"
	GenesisBalanceKey = "genesis_balance"
)

var InitCommandHooks = func(cmd *InitCommand) []pm.Hook {
	genesisOperationHandlers := map[string]process.HookHandlerGenesisOperations{
		"genesis-currencies": GenesisOperationsHandlerGenesisCurrencies,
	}

	for k, v := range process.DefaultHookHandlersGenesisOperations {
		genesisOperationHandlers[k] = v
	}

	return []pm.Hook{
		pm.NewHook(pm.HookPrefixPre, process.ProcessNameProposalProcessor,
			"initialize_proposal_processor", cmd.hookInitializeProposalProcessor).SetOverride(true),
		pm.NewHook(pm.HookPrefixPost, process.ProcessNameConfig,
			process.HookNameConfigGenesisOperations, process.HookGenesisOperationFunc(genesisOperationHandlers)).
			SetOverride(true),
	}
}

type InitCommand struct {
	*BaseNodeCommand
	*mitumcmds.InitCommand
}

func NewInitCommand(dryrun bool) (InitCommand, error) {
	co := mitumcmds.NewInitCommand(dryrun)
	cmd := InitCommand{
		InitCommand:     &co,
		BaseNodeCommand: NewBaseNodeCommand(co.Logging),
	}

	ps, err := cmd.BaseProcesses(co.Processes())
	if err != nil {
		return cmd, err
	}

	hooks := InitCommandHooks(&cmd)
	for i := range hooks {
		if err := hooks[i].Add(ps); err != nil {
			return cmd, err
		}
	}

	_ = cmd.SetProcesses(ps)

	return cmd, nil
}

func (*InitCommand) hookInitializeProposalProcessor(ctx context.Context) (context.Context, error) {
	var oprs *hint.Hintmap
	if err := process.LoadOperationProcessorsContextValue(ctx, &oprs); err != nil {
		if !xerrors.Is(err, util.ContextValueNotFoundError) {
			return ctx, err
		}
	}

	if oprs == nil {
		oprs = hint.NewHintmap()

		ctx = context.WithValue(ctx, process.ContextValueOperationProcessors, oprs)
	}

	return ctx, nil
}

func GenesisOperationsHandlerGenesisCurrencies(
	ctx context.Context,
	m map[string]interface{},
) (operation.Operation, error) {
	var conf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
		return nil, err
	}

	var de *GenesisCurrenciesDesign
	if b, err := yaml.Marshal(m); err != nil {
		return nil, err
	} else if err := yaml.Unmarshal(b, &de); err != nil {
		return nil, err
	}

	if err := de.IsValid(nil); err != nil {
		return nil, err
	}

	cds := make([]currency.CurrencyDesign, len(de.Currencies))
	for i := range de.Currencies {
		c := de.Currencies[i]

		j, err := loadCurrencyDesign(*c, de.AccountKeys.Address)
		if err != nil {
			return nil, err
		}
		cds[i] = j
	}

	if op, err := currency.NewGenesisCurrencies(
		conf.Privatekey(),
		de.AccountKeys.Keys,
		cds,
		conf.NetworkID(),
	); err != nil {
		return nil, err
	} else if err := op.IsValid(conf.NetworkID()); err != nil {
		return nil, err
	} else {
		return op, nil
	}
}

func loadCurrencyDesign(de CurrencyDesign, ga base.Address) (currency.CurrencyDesign, error) {
	j, err := loadGenesisCurrenciesFeeer(*de.Feeer, ga)
	if err != nil {
		return currency.CurrencyDesign{}, err
	}
	po := currency.NewCurrencyPolicy(de.NewAccountMinBalance, j)

	cd := currency.NewCurrencyDesign(de.Balance, nil, po)
	if err := cd.IsValid(nil); err != nil {
		return currency.CurrencyDesign{}, err
	}

	return cd, nil
}

func loadGenesisCurrenciesFeeer(de FeeerDesign, ga base.Address) (currency.Feeer, error) {
	var feeer currency.Feeer
	switch de.Type {
	case currency.FeeerNil, "":
		return currency.NewNilFeeer(), nil
	case currency.FeeerFixed:
		feeer = currency.NewFixedFeeer(ga, de.Extras["fixed_amount"].(currency.Big))
	case currency.FeeerRatio:
		var max currency.Big
		if i, found := de.Extras["ratio_max"]; !found {
			max = currency.UnlimitedMaxFeeAmount
		} else {
			max = i.(currency.Big)
		}

		feeer = currency.NewRatioFeeer(
			ga,
			de.Extras["ratio_ratio"].(float64),
			de.Extras["ratio_min"].(currency.Big),
			max,
		)
	default:
		return nil, xerrors.Errorf("unknown type of feeer, %q", de.Type)
	}

	if err := feeer.IsValid(nil); err != nil {
		return nil, err
	}

	return feeer, nil
}
