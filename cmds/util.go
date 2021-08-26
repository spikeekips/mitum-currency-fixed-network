package cmds

import (
	"fmt"
	"io"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func PrettyPrint(out io.Writer, pretty bool, i interface{}) {
	var b []byte
	if pretty {
		b = jsonenc.MustMarshalIndent(i)
	} else {
		b = jsonenc.MustMarshal(i)
	}

	_, _ = fmt.Fprintln(out, string(b))
}
