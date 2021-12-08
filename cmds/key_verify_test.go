package cmds

import (
	"bytes"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/util"
)

type testVerifyKeyCommand struct {
	suite.Suite
}

func (t *testVerifyKeyCommand) TestSingleKey() {
	cli := NewVerifyKeyCommand()
	parser, err := kong.New(&cli, cmds.LogVars, cmds.PprofVars)
	t.NoError(err)

	_, err = parser.Parse([]string{"KzFERQKNQbPA8cdsX5tCiCZvR4KgBou41cgtPk69XueFbaEjrczbmpr"})
	t.NoError(err)

	var buf bytes.Buffer
	cli.Out = &buf

	t.NoError(cli.Run(util.Version("0.1.1")))

	t.Equal(`privatekey hint: mpr
     privatekey: KzFERQKNQbPA8cdsX5tCiCZvR4KgBou41cgtPk69XueFbaEjrczbmpr
 publickey hint: mpu
      publickey: zzeo6WAS4uqwCss4eRibtLnYHqJM21zhzPbKWQVPttxWmpu
`, buf.String())
}

func TestVerifyKeyCommand(t *testing.T) {
	suite.Run(t, new(testVerifyKeyCommand))
}
