package digest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"golang.org/x/xerrors"
)

func (hd *Handlers) SetSend(f func(interface{}) (seal.Seal, error)) *Handlers {
	hd.send = f

	return hd
}

func (hd *Handlers) handleSend(w http.ResponseWriter, r *http.Request) {
	if hd.send == nil {
		hd.notSupported(w, nil)

		return
	}

	body := &bytes.Buffer{}
	if _, err := io.Copy(body, r.Body); err != nil {
		hd.problemWithError(w, err, http.StatusInternalServerError)

		return
	}

	var hal Hal
	var v []json.RawMessage
	if err := jsonenc.Unmarshal(body.Bytes(), &v); err != nil {
		if hinter, err := hd.enc.DecodeByHint(body.Bytes()); err != nil {
			hd.problemWithError(w, err, http.StatusInternalServerError)

			return
		} else if h, err := hd.sendSeal(hinter); err != nil {
			hd.problemWithError(w, err, http.StatusInternalServerError)

			return
		} else {
			hal = h
		}
	} else { // NOTE only support operation.Operation
		ops := make([]operation.Operation, len(v))
		for i := range v {
			if hinter, err := hd.enc.DecodeByHint(v[i]); err != nil {
				hd.problemWithError(w, err, http.StatusBadRequest)

				return
			} else if op, ok := hinter.(operation.Operation); !ok {
				hd.problemWithError(w, xerrors.Errorf("unsupported message type, %T", hinter), http.StatusBadRequest)

				return
			} else {
				ops[i] = op
			}
		}

		if h, err := hd.sendSeal((operation.BaseSeal{}).SetOperations(ops)); err != nil {
			hd.problemWithError(w, err, http.StatusInternalServerError)

			return
		} else {
			hal = h
		}
	}

	hd.writeHal(w, hal, http.StatusOK)
}

func (hd *Handlers) sendSeal(v interface{}) (Hal, error) {
	if sl, err := hd.send(v); err != nil {
		return nil, err
	} else {
		return hd.buildSealHal(sl)
	}
}

func (hd *Handlers) buildSealHal(sl seal.Seal) (Hal, error) {
	var hal Hal = NewBaseHal(sl, HalLink{})
	if t, ok := sl.(operation.Seal); ok {
		for i := range t.Operations() {
			op := t.Operations()[i]
			if h, err := hd.combineURL(HandlerPathOperation, "hash", op.Fact().Hash().String()); err != nil {
				return nil, err
			} else {
				hal.AddLink(fmt.Sprintf("operation:%d", i), NewHalLink(h, nil))
			}
		}
	}

	return hal, nil
}
