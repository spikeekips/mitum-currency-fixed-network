package currency

import "github.com/spikeekips/mitum/util/isvalid"

var MaxMemoSize = 100 // TODO should be managed by policy

func IsValidMemo(s string) error {
	if len(s) > MaxMemoSize {
		return isvalid.InvalidError.Errorf("memo over max size, %d > %d", len(s), MaxMemoSize)
	}

	return nil
}

type MemoBSONUnpacker struct {
	Memo string `bson:"memo"`
}

type MemoJSONUnpacker struct {
	Memo string `json:"memo"`
}
