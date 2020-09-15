package cmds

import (
	"bytes"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/stretchr/testify/suite"
)

type testVerifyKeyCommand struct {
	suite.Suite
}

func (t *testVerifyKeyCommand) TestSingleKey() {
	cli := &VerifyKeyCommand{}
	parser, err := kong.New(cli)
	t.NoError(err)

	_, err = parser.Parse([]string{"KzbpUPUhHPxHnaQZndkfQvKoj2MxLjhjQLAGs42kuM3UEsgFNUoX-0112:0.0.1"})
	t.NoError(err)

	var buf bytes.Buffer
	cli.o = &buf

	t.NoError(cli.Run(nil, util.Version("0.1.1"), logging.NilLogger))

	t.Equal(`privatekey hint: hint{type="btc-privatekey" code="0112" version="0.0.1"}
     privatekey: KzbpUPUhHPxHnaQZndkfQvKoj2MxLjhjQLAGs42kuM3UEsgFNUoX-0112:0.0.1
 publickey hint: hint{type="btc-publickey" code="0113" version="0.0.1"}
      publickey: mbxYSTvbpdN7ANWEav536HzDivVu9tqGgKzZjcXJLYKY-0113:0.0.1
`, string(buf.Bytes()))
}

func TestVerifyKeyCommand(t *testing.T) {
	suite.Run(t, new(testVerifyKeyCommand))
}
