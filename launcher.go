package mc

import (
	"fmt"
	"net/url"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type Launcher struct {
	*logging.Logging
	*launcher.Launcher
	design *NodeDesign
}

func NewLauncherFromDesign(design *NodeDesign, version util.Version) (*Launcher, error) {
	nr := &Launcher{design: design}

	if ca, err := NewAddress(design.Address); err != nil {
		return nil, err
	} else if bn, err := launcher.NewLauncher(ca, design.Privatekey(), design.NetworkID(), version); err != nil {
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
		nr.Launcher.Initialize,
	} {
		if err := f(); err != nil {
			return err
		}
	}

	return nil
}

func (nr *Launcher) attachStorage() error {
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
			return ctx.Str("address", r.Address)
		})

		l.Debug().Msg("trying to create remote node")

		var n *isaac.RemoteNode
		if ca, err := NewAddress(r.Address); err != nil {
			return err
		} else {
			n = isaac.NewRemoteNode(ca, r.Publickey())
		}

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
