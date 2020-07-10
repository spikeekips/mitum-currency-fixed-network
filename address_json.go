package mc

import (
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type AddressJSONPacker struct {
	jsonenc.HintedHead
	A string `json:"address"`
}

func (ca Address) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(AddressJSONPacker{
		HintedHead: jsonenc.NewHintedHead(ca.Hint()),
		A:          ca.String(),
	})
}

type AddressJSONUnpacker struct {
	A string `json:"address"`
}

func (ca *Address) UnmarshalJSON(b []byte) error {
	var uca AddressJSONUnpacker
	if err := util.JSON.Unmarshal(b, &uca); err != nil {
		return err
	}

	return ca.unpack(nil, uca.A)
}
