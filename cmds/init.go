package cmds

import (
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

var (
	encs *encoder.Encoders
	jenc *jsonenc.Encoder
)

func init() {
	// NOTE mitum internal version should be 'v0.0.1-stable'
	if mitumcmds.InternalVersion != "v0.0.1-stable" {
		panic("mitum should be v0.0.1-stable")
	}

	cmd := mitumcmds.NewBaseCommand("")
	if i, err := cmd.LoadEncoders(Types, Hinters); err != nil {
		panic(err)
	} else {
		encs = i
		jenc = cmd.JSONEncoder()
	}
}
