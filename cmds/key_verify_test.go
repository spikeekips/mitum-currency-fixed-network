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

	_, err = parser.Parse([]string{"KzbpUPUhHPxHnaQZndkfQvKoj2MxLjhjQLAGs42kuM3UEsgFNUoX:btc-priv-v0.0.1"})
	t.NoError(err)

	var buf bytes.Buffer
	cli.out = &buf

	t.NoError(cli.Run(util.Version("0.1.1")))

	t.Equal(`privatekey hint: btc-priv-v0.0.1
     privatekey: KzbpUPUhHPxHnaQZndkfQvKoj2MxLjhjQLAGs42kuM3UEsgFNUoX:btc-priv-v0.0.1
 publickey hint: btc-pub-v0.0.1
      publickey: mbxYSTvbpdN7ANWEav536HzDivVu9tqGgKzZjcXJLYKY:btc-pub-v0.0.1
`, buf.String())
}

func TestVerifyKeyCommand(t *testing.T) {
	suite.Run(t, new(testVerifyKeyCommand))
}
