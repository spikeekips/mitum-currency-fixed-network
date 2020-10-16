package digest

import (
	"fmt"
	"strings"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

func IsAccountState(st state.State) (currency.Account, bool, error) {
	if !currency.IsStateAccountKey(st.Key()) {
		return currency.Account{}, false, nil
	}

	if ac, err := currency.LoadStateAccountValue(st); err != nil {
		return currency.Account{}, false, err
	} else {
		return ac, true, nil
	}
}

func IsBalanceState(st state.State) (currency.Amount, bool, error) {
	if !currency.IsStateBalanceKey(st.Key()) {
		return currency.NilAmount, false, nil
	}

	if am, err := currency.StateAmountValue(st); err != nil {
		return currency.NilAmount, false, err
	} else {
		return am, true, nil
	}
}

func parseHeightFromPath(s string) (base.Height, error) {
	s = strings.TrimSpace(s)

	if len(s) < 1 {
		return base.NilHeight, xerrors.Errorf("empty height")
	} else if len(s) > 1 && strings.HasPrefix(s, "0") {
		return base.NilHeight, xerrors.Errorf("invalid height, %q", s)
	}

	return base.NewHeightFromString(s)
}

func parseHashFromPath(s string) (valuehash.Hash, error) {
	s = strings.TrimSpace(s)
	if len(s) < 1 {
		return nil, xerrors.Errorf("empty hash")
	}

	if h := valuehash.NewBytesFromString(s); h.Empty() {
		return nil, xerrors.Errorf("empty hash")
	} else {
		return h, nil
	}
}

func parseOffsetQuery(s string) string {
	return strings.TrimSpace(s)
}

func stringOffsetQuery(offset string) string {
	return fmt.Sprintf("offset=%s", offset)
}

func parseBoolQuery(s string) bool {
	return s == "1"
}

func stringBoolQuery(key string, v bool) string { // nolint:unparam
	if v {
		return fmt.Sprintf("%s=1", key)
	}

	return ""
}

func addQueryValue(base, s string) string {
	if len(s) < 1 {
		return base
	}

	if !strings.Contains(base, "?") {
		return base + "?" + s
	}

	return base + "&" + s
}
