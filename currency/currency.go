package currency

import (
	"regexp"

	"github.com/pkg/errors"
)

var (
	MinLengthCurrencyID = 3
	MaxLengthCurrencyID = 10
	ReValidCurrencyID   = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_\.\!\$\*\+\@]*[A-Z0-9]$`)
)

type CurrencyID string

func (cid CurrencyID) Bytes() []byte {
	return []byte(cid)
}

func (cid CurrencyID) String() string {
	return string(cid)
}

func (cid CurrencyID) IsValid([]byte) error {
	if l := len(cid); l < MinLengthCurrencyID || l > MaxLengthCurrencyID {
		return errors.Errorf(
			"invalid length of currency id, %d <= %d <= %d", MinLengthCurrencyID, l, MaxLengthCurrencyID)
	} else if !ReValidCurrencyID.Match([]byte(cid)) {
		return errors.Errorf("wrong currency id, %q", cid)
	}

	return nil
}
