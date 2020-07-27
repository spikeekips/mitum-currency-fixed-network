package currency

import "golang.org/x/xerrors"

var MaxMemoSize = 100 // TODO should be managed by policy

func IsValidMemo(s string) error {
	if len(s) > MaxMemoSize {
		return xerrors.Errorf("memo over max size, %d > %d", len(s), MaxMemoSize)
	}

	return nil
}

type MemoBSONUnpacker struct {
	Memo string `bson:"memo"`
}

type MemoJSONUnpacker struct {
	Memo string `json:"memo"`
}
