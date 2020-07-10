package mc

import (
	"fmt"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
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
	if n := len(keys); n < 0 {
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

	if len(keys) == 1 {
		return NewAddress(keys[0].Key().Raw())
	}

	skeys := make([]string, len(keys))
	for i := range keys {
		skeys[i] = fmt.Sprintf("%s:%d", keys[i].Key().Raw(), keys[i].Weight())
	}

	sort.Strings(skeys)

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

func (ca Address) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(ca.String())
}

func (ca *Address) UnmarshalJSON(b []byte) error {
	var a string
	if err := util.JSON.Unmarshal(b, &a); err != nil {
		return err
	}

	*ca = Address(a)

	return nil
}

func (ca Address) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, ca.String()), nil
}

func (ca *Address) UnmarshalBSONValue(t bsontype.Type, b []byte) error {
	if t != bsontype.String {
		return xerrors.Errorf("invalid marshaled type for Address, %v", t)
	}

	s, _, ok := bsoncore.ReadString(b)
	if !ok {
		return xerrors.Errorf("can not read string")
	}

	*ca = Address(s)

	return nil
}

func (ca Address) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	if !verbose {
		return e.Str(key, ca.String())
	}

	return e.Str(key, ca.String())
}
