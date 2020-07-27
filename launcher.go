package mc

import (
	"fmt"
	"net/url"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type Launcher struct {
	*logging.Logging
	*launcher.Launcher
	design *launcher.NodeDesign
}

func NewLauncherFromDesign(design *launcher.NodeDesign, version util.Version) (*Launcher, error) {
	nr := &Launcher{design: design}

	if bn, err := launcher.NewLauncher(design, version); err != nil {
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

func (nr *Launcher) Design() *launcher.NodeDesign {
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

	if st, err := launcher.LoadStorage(nr.design.Storage, nr.Encoders()); err != nil {
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
	if qs, err := launcher.LoadNetworkServer(nd.Bind, nd.PublishURL(), nr.Encoders()); err != nil {
		return err
	} else {
		_ = nr.SetNetwork(qs)
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
		_ = nr.SetNodeChannel(ch)
		_ = nr.Localstate().Node().SetChannel(ch)
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

		n := isaac.NewRemoteNode(r.Address(), r.Publickey())
		if ch, err := launcher.LoadNodeChannel(r.NetworkURL(), nr.Encoders()); err != nil {
			return err
		} else {
			_ = n.SetChannel(ch)
		}
		l.Debug().Msg("created")

		nodes[i] = n
	}

	return nr.Localstate().Nodes().Add(nodes...)
}

func (nr *Launcher) attachSuffrage() error {
	l := nr.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("target", "suffrage")
	})
	l.Debug().Msg("trying to attach")

	if sf, err := nr.design.Component.Suffrage.New(nr.Localstate(), nr.Encoders()); err != nil {
		return xerrors.Errorf("failed to create new suffrage component: %w", err)
	} else {
		l.Debug().
			Str("type", nr.design.Component.Suffrage.Type).
			Interface("info", nr.design.Component.Suffrage.Info).
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

	if pp, err := nr.design.Component.ProposalProcessor.New(nr.Localstate(), nr.Suffrage()); err != nil {
		return xerrors.Errorf("failed to create new proposal processor component: %w", err)
	} else {
		l.Debug().
			Str("type", nr.design.Component.ProposalProcessor.Type).
			Interface("info", nr.design.Component.ProposalProcessor.Info).
			Msg("proposal processor loaded")

		_ = nr.SetProposalProcessor(pp)
	}

	l.Debug().Msg("attached")

	return nil
}
