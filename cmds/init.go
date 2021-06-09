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
	cmd := mitumcmds.NewBaseCommand("")
	if i, err := cmd.LoadEncoders(Types, Hinters); err != nil {
		panic(err)
	} else {
		encs = i
		jenc = cmd.JSONEncoder()
	}
}
