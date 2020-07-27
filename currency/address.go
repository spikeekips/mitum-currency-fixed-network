package currency

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	AddressType = hint.MustNewType(0xa0, 0x00, "mitum-currency-address")
	AddressHint = hint.MustHint(AddressType, "0.0.1")
)

var EmptyAddress = Address("")

type Address string

func NewAddress(name string) (Address, error) {
	ca := Address(name)

	return ca, ca.IsValid(nil)
}

func NewAddressFromKeys(keys []Key) (Address, error) {
	if n := len(keys); n < 1 {
		return EmptyAddress, xerrors.Errorf("empty keys for Address")
	} else {
		for i := range keys {
			k := keys[i]
			if err := k.IsValid(nil); err != nil {
				return EmptyAddress, xerrors.Errorf("invalid key found: %w", err)
			} else if _, ok := k.Key().(key.Publickey); !ok {
				return EmptyAddress, xerrors.Errorf("key should be key.Publickey; %T found", k)
			}
		}
	}

	skeys := make([]string, len(keys))
	for i := range keys {
		skeys[i] = fmt.Sprintf("%s:%d", keys[i].Key().String(), keys[i].Weight())
	}

	if len(keys) > 1 {
		sort.Strings(skeys)
	}

	return NewAddress(valuehash.NewSHA256([]byte(strings.Join(skeys, ","))).String())
}

func (ca Address) String() string {
	return string(ca)
}

func (ca Address) Hint() hint.Hint {
	return AddressHint
}

func (ca Address) IsValid([]byte) error {
	if s := strings.TrimSpace(ca.String()); len(s) < 1 {
		return xerrors.Errorf("empty address")
	}

	return nil
}

func (ca Address) Equal(a base.Address) bool {
	if ca.Hint().Type() != a.Hint().Type() {
		return false
	}

	return ca == a.(Address)
}

func (ca Address) Bytes() []byte {
	return []byte(ca)
}

func (ca Address) MarshalText() ([]byte, error) {
	return []byte(hint.HintedString(ca.Hint(), ca.String())), nil
}

func (ca *Address) UnmarshalText(b []byte) error {
	if a, err := NewAddress(string(b)); err != nil {
		return err
	} else {
		*ca = a

		return nil
	}
}

func (ca Address) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	if !verbose {
		return e.Str(key, ca.String())
	}

	return e.Str(key, ca.String())
}
