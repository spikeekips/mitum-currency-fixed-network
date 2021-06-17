// +build mongodb

package digest

import (
	"io"
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testHandlerNodeInfo struct {
	baseTestHandlers
}

func (t *testHandlerNodeInfo) TestBasic() {
	st, _ := t.Database()

	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	local := base.RandomNode("n0").SetURL("quic://local")

	na1, err := base.NewStringAddress("n1")
	t.NoError(err)
	n1 := base.NewBaseNodeV0(na1, key.MustNewBTCPrivatekey().Publickey(), "quic://n1")

	na2, err := base.NewStringAddress("n2")
	t.NoError(err)
	n2 := base.NewBaseNodeV0(na2, key.MustNewBTCPrivatekey().Publickey(), "quic://n2")

	nodes := []base.Node{n1, n2}
	policy := map[string]interface{}{"showme": 1}

	ni := network.NewNodeInfoV0(
		local,
		t.networkID,
		base.StateBooting,
		blk.Manifest(),
		util.Version("0.1.1"),
		"quic://local",
		policy,
		nodes,
		nil,
	)

	handlers := t.handlers(st, DummyCache{})

	handlers.SetNodeInfoHandler(func() (network.NodeInfo, error) {
		return NewNodeInfo(ni), nil
	})

	self, err := handlers.router.Get("root").URL()
	t.NoError(err)

	w := t.requestOK(handlers, "GET", self.Path, nil)

	b, err := io.ReadAll(w.Result().Body)
	t.NoError(err)

	hal := t.loadHal(b)

	uni, err := network.DecodeNodeInfo(hal.RawInterface(), t.JSONEnc)
	t.NoError(err)

	t.compareNodeInfo(ni, uni)
}

func (t *testHandlerNodeInfo) compareNodeInfo(a, b network.NodeInfo) {
	t.True(a.Address().Equal(b.Address()))
	t.True(a.Publickey().Equal(b.Publickey()))
	t.Equal(a.NetworkID(), b.NetworkID())
	t.Equal(a.Version(), b.Version())
	t.Equal(a.URL(), b.URL())

	t.Equal(len(a.Policy()), len(b.Policy()))
	{
		ab, err := jsonenc.Marshal(a.Policy())
		t.NoError(err)
		bb, err := jsonenc.Marshal(b.Policy())
		t.NoError(err)
		t.Equal(ab, bb)
	}

	t.Equal(len(a.Nodes()), len(b.Nodes()))
	for i := range a.Nodes() {
		an := a.Nodes()[i]
		bn := b.Nodes()[i]

		t.True(an.Address().Equal(bn.Address()))
		t.True(an.Publickey().Equal(bn.Publickey()))
		t.Equal(an.URL(), bn.URL())
	}
}

func TestHandlerNodeInfo(t *testing.T) {
	suite.Run(t, new(testHandlerNodeInfo))
}
