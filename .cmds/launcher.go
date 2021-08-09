package cmds

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/pkg/errors"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum-currency/digest"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/logging"
)

var (
	GenesisAccountKey = "genesis_account"
	GenesisBalanceKey = "genesis_balance"
)

type Launcher struct {
	sync.RWMutex
	*logging.Logging
	*launcher.Launcher
	design         *NodeDesign
	genesisAccount currency.Account
	genesisBalance currency.Amount
}

func NewLauncherFromDesign(design *NodeDesign, version util.Version) (*Launcher, error) {
	nr := &Launcher{design: design}

	if bn, err := launcher.NewLauncher(design.NodeDesign, version); err != nil {
		return nil, err
	} else {
		nr.Launcher = bn
	}

	nr.Logging = logging.NewLogging(func(c logging.Context) logging.Emitter {
		return c.Str("module", "mitum-currency-node-runner")
	})
	return nr, nil
}

func (nr *Launcher) SetLogger(l logging.Logger) logging.Logger {
	_ = nr.Launcher.SetLogger(l)
	_ = nr.Logging.SetLogger(l)

	return nr.Log()
}

func (nr *Launcher) Design() *NodeDesign {
	return nr.design
}

func (nr *Launcher) Initialize() error {
	for _, f := range []func() error{
		nr.attachStorage,
		nr.attachNetwork,
		nr.attachNodeChannel,
		nr.attachRemoteNodes,
		nr.attachSuffrage,
		nr.attachProposalProcessor,
		nr.Launcher.Initialize,
	} {
		if err := f(); err != nil {
			return err
		}
	}

	return nil
}

func (nr *Launcher) AttachStorage() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "storage")
	})
	l.Debug().Msg("trying to attach")

	var ca cache.Cache
	if len(nr.design.Component.StorageCache()) > 0 {
		if c, err := cache.NewCacheFromURI(nr.design.Component.StorageCache()); err != nil {
			return err
		} else {
			ca = c
		}
	}

	if st, err := launcher.LoadStorage(nr.design.Storage, nr.Encoders(), ca); err != nil {
		return err
	} else {
		_ = nr.SetStorage(st)
	}

	l.Debug().Msg("attached")

	return nil
}

func (nr *Launcher) attachStorage() error {
	if nr.Storage() != nil {
		return nil
	}

	return nr.AttachStorage()
}

func (nr *Launcher) attachNetwork() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "network")
	})
	l.Debug().Msg("trying to attach")

	nd := nr.design.Network
	if qs, err := launcher.LoadNetworkServer(nd.Bind().Host, nd.PublishURL(), nr.Encoders()); err != nil {
		return err
	} else {
		_ = nr.SetNetwork(qs)
		qs.SetNodeInfoHandler(nr.nodeInfoHandler)

		_ = nr.SetPublichURL(nr.design.Network.PublishURL().String())
	}

	l.Debug().Msg("attached")

	return nil
}

func (nr *Launcher) attachNodeChannel() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "node-channel")
	})
	l.Debug().Msg("trying to attach")

	nu := new(url.URL)
	*nu = *nr.design.Network.PublishURL()
	nu.Host = fmt.Sprintf("localhost:%s", nu.Port())

	if ch, err := launcher.LoadNodeChannel(nu, nr.Encoders()); err != nil {
		return err
	} else {
		if l, ok := ch.(logging.SetLogger); ok {
			_ = l.SetLogger(nr.Log())
		}

		_ = nr.SetNodeChannel(ch)
		_ = nr.Local().Node().SetChannel(ch)
	}

	l.Debug().Msg("attached")

	return nil
}

func (nr *Launcher) attachRemoteNodes() error {
	nodes := make([]network.Node, len(nr.design.Nodes))

	for i, r := range nr.design.Nodes {
		r := r
		l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
			return ctx.Hinted("address", r.Address())
		})

		l.Debug().Msg("trying to create remote node")

		if n, err := r.NetworkNode(nr.Encoders()); err != nil {
			return err
		} else {
			if l, ok := n.Channel().(logging.SetLogger); ok {
				_ = l.SetLogger(nr.Log())
			}

			nodes[i] = n
		}
		l.Debug().Msg("created")
	}

	return nr.Local().Nodes().Add(nodes...)
}

func (nr *Launcher) attachSuffrage() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "suffrage")
	})
	l.Debug().Msg("trying to attach")

	if sf, err := nr.design.Component.Suffrage().New(nr.Local(), nr.Encoders()); err != nil {
		return errors.Wrap(err, "failed to create new suffrage component")
	} else {
		l.Debug().
			Str("type", nr.design.Component.Suffrage().Type).
			Interface("info", nr.design.Component.Suffrage().Info).
			Msg("suffrage loaded")

		_ = nr.SetSuffrage(sf)
	}

	l.Debug().Msg("attached")

	return nil
}

func (nr *Launcher) attachProposalProcessor() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "proposal-processor")
	})
	l.Debug().Msg("trying to attach")

	if pp, err := nr.design.Component.ProposalProcessor().New(nr.Local(), nr.Suffrage()); err != nil {
		return errors.Wrap(err, "failed to create new proposal processor component")
	} else {
		l.Debug().
			Str("type", nr.design.Component.ProposalProcessor().Type).
			Interface("info", nr.design.Component.ProposalProcessor().Info).
			Msg("proposal processor loaded")

		_ = nr.SetProposalProcessor(pp)
	}

	l.Debug().Msg("attached")

	return nil
}

func (nr *Launcher) genesisInfo() (currency.Account, currency.Amount, bool) {
	nr.RLock()
	defer nr.RUnlock()

	return nr.genesisAccount, nr.genesisBalance, nr.genesisBalance.Compare(currency.ZeroAmount) > 0
}

func (nr *Launcher) setGenesisInfo(ac currency.Account, balance currency.Amount) {
	nr.Lock()
	defer nr.Unlock()

	nr.genesisAccount = ac
	nr.genesisBalance = balance
}

func (nr *Launcher) nodeInfoHandler() (network.NodeInfo, error) {
	var ga currency.Account
	var gb currency.Amount
	if ac, ba, exist := nr.genesisInfo(); !exist {
		nr.Log().Debug().Msg("genesis info not found")
	} else {
		ga = ac
		gb = ba
	}

	if i, err := nr.NodeInfo(); err != nil {
		return nil, err
	} else if ni, ok := i.(network.NodeInfoV0); !ok {
		return nil, errors.Errorf("unsupported NodeInfo, %T", i)
	} else {
		return digest.NewNodeInfo(ni, nr.Design().FeeAmount, ga, gb), nil
	}
}
