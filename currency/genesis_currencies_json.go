package currency

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type GenesisCurrenciesFactJSONPacker struct {
	jsonenc.HintedHead
	H  valuehash.Hash   `json:"hash"`
	TK []byte           `json:"token"`
	GK key.Publickey    `json:"genesis_node_key"`
	KS Keys             `json:"keys"`
	CS []CurrencyDesign `json:"currencies"`
}

func (fact GenesisCurrenciesFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(GenesisCurrenciesFactJSONPacker{
		HintedHead: jsonenc.NewHintedHead(fact.Hint()),
		H:          fact.h,
		TK:         fact.token,
		GK:         fact.genesisNodeKey,
		KS:         fact.keys,
		CS:         fact.cs,
	})
}

type GenesisCurrenciesFactJSONUnpacker struct {
	H  valuehash.Bytes      `json:"hash"`
	TK []byte               `json:"token"`
	GK key.PublickeyDecoder `json:"genesis_node_key"`
	KS json.RawMessage      `json:"keys"`
	CS json.RawMessage      `json:"currencies"`
}

func (fact *GenesisCurrenciesFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ufact GenesisCurrenciesFactJSONUnpacker
	if err := jsonenc.Unmarshal(b, &ufact); err != nil {
		return errors.Wrap(err, "failed to unmarshal GenesisCurrenciesFact")
	}

	return fact.unpack(enc, ufact.H, ufact.TK, ufact.GK, ufact.KS, ufact.CS)
}

func (op GenesisCurrencies) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(op.BaseOperation)
}

func (op *GenesisCurrencies) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ubo operation.BaseOperation
	if err := ubo.UnpackJSON(b, enc); err != nil {
		return err
	}

	*op = GenesisCurrencies{BaseOperation: ubo}

	return nil
}
