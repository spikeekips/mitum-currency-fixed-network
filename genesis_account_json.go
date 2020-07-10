package mc

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type GenesisAccountFactJSONPacker struct {
	jsonenc.HintedHead
	H  valuehash.Hash `json:"hash"`
	TK []byte         `json:"token"`
	GK key.Publickey  `json:"genesis_node_key"`
	KS Keys           `json:"keys"`
	AM Amount         `json:"amount"`
}

func (gaf GenesisAccountFact) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(GenesisAccountFactJSONPacker{
		HintedHead: jsonenc.NewHintedHead(gaf.Hint()),
		H:          gaf.h,
		TK:         gaf.token,
		GK:         gaf.genesisNodeKey,
		KS:         gaf.keys,
		AM:         gaf.amount,
	})
}

type GenesisAccountFactJSONUnpacker struct {
	H  valuehash.Bytes `json:"hash"`
	TK []byte          `json:"token"`
	GK key.KeyDecoder  `json:"genesis_node_key"`
	KS json.RawMessage `json:"keys"`
	AM Amount          `json:"amount"`
}

func (gaf *GenesisAccountFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uca GenesisAccountFactJSONUnpacker
	if err := util.JSON.Unmarshal(b, &uca); err != nil {
		return err
	}

	return gaf.unpack(enc, uca.H, uca.TK, uca.GK, uca.KS, uca.AM)
}

func (ga GenesisAccount) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(ga.BaseOperation)
}

func (ga *GenesisAccount) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ubo operation.BaseOperation
	if err := ubo.UnpackJSON(b, enc); err != nil {
		return err
	}

	*ga = GenesisAccount{BaseOperation: ubo}

	return nil
}
