package currency

import (
	"bytes"
	"sort"

	"github.com/pkg/errors"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	KeyType  = hint.Type("mitum-currency-key")
	KeyHint  = hint.NewHint(KeyType, "v0.0.1")
	KeysType = hint.Type("mitum-currency-keys")
	KeysHint = hint.NewHint(KeysType, "v0.0.1")
)

var (
	MaxKeyInKeys int
	maxKeyInKeys uint = 10
)

func init() {
	MaxKeyInKeys = int(maxKeyInKeys)
}

type Key struct {
	k key.Publickey
	w uint
}

func NewKey(k key.Publickey, w uint) (Key, error) {
	ky := Key{k: k, w: w}

	return ky, ky.IsValid(nil)
}

func (ky Key) IsValid([]byte) error {
	if ky.w < 1 || ky.w > 100 {
		return errors.Errorf("invalid key weight, 1 <= weight <= 100")
	}

	return ky.k.IsValid(nil)
}

func (ky Key) Weight() uint {
	return ky.w
}

func (ky Key) Key() key.Publickey {
	return ky.k
}

func (Key) Hint() hint.Hint {
	return KeyHint
}

func (ky Key) Bytes() []byte {
	return util.ConcatBytesSlice([]byte(ky.k.String()), util.UintToBytes(ky.w))
}

func (ky Key) Equal(b Key) bool {
	if ky.w != b.w {
		return false
	}

	if !ky.k.Equal(b.k) {
		return false
	}

	return true
}

type Keys struct {
	h         valuehash.Hash
	keys      []Key
	threshold uint
}

func NewKeys(keys []Key, threshold uint) (Keys, error) {
	ks := Keys{keys: keys, threshold: threshold}
	h, err := ks.GenerateHash()
	if err != nil {
		return Keys{}, err
	}
	ks.h = h

	return ks, ks.IsValid(nil)
}

func (Keys) Hint() hint.Hint {
	return KeysHint
}

func (ks Keys) Hash() valuehash.Hash {
	return ks.h
}

func (ks Keys) GenerateHash() (valuehash.Hash, error) {
	return valuehash.NewSHA256(ks.Bytes()), nil
}

func (ks Keys) Bytes() []byte {
	bs := make([][]byte, len(ks.keys)+1)

	// NOTE sorted by Key.Key()
	sort.Slice(ks.keys, func(i, j int) bool {
		return bytes.Compare(ks.keys[i].Key().Bytes(), ks.keys[j].Key().Bytes()) < 0
	})
	for i := range ks.keys {
		bs[i] = ks.keys[i].Bytes()
	}

	bs[len(ks.keys)] = util.UintToBytes(ks.threshold)

	return util.ConcatBytesSlice(bs...)
}

func (ks Keys) IsValid([]byte) error {
	if ks.threshold < 1 || ks.threshold > 100 {
		return errors.Errorf("invalid threshold, %d, should be 1 <= threshold <= 100", ks.threshold)
	}

	if err := ks.h.IsValid(nil); err != nil {
		return err
	}

	if n := len(ks.keys); n < 1 {
		return errors.Errorf("empty keys")
	} else if n > MaxKeyInKeys {
		return errors.Errorf("keys over %d, %d", MaxKeyInKeys, n)
	}

	m := map[string]struct{}{}
	for i := range ks.keys {
		k := ks.keys[i]
		if err := k.IsValid(nil); err != nil {
			return err
		} else if _, found := m[k.Key().String()]; found {
			return errors.Errorf("duplicated keys found")
		} else {
			m[k.Key().String()] = struct{}{}
		}
	}

	var totalWeight uint
	for i := range ks.keys {
		totalWeight += ks.keys[i].Weight()
	}

	if totalWeight < ks.threshold {
		return errors.Errorf("sum of weight under threshold, %d < %d", totalWeight, ks.threshold)
	}

	if h, err := ks.GenerateHash(); err != nil {
		return err
	} else if !ks.h.Equal(h) {
		return errors.Errorf("hash not matched")
	}

	return nil
}

func (ks Keys) Threshold() uint {
	return ks.threshold
}

func (ks Keys) Keys() []Key {
	return ks.keys
}

func (ks Keys) Key(k key.Publickey) (Key, bool) {
	for i := range ks.keys {
		ky := ks.keys[i]
		if ky.Key().Equal(k) {
			return ky, true
		}
	}

	return Key{}, false
}

func (ks Keys) Equal(b Keys) bool {
	if ks.threshold != b.threshold {
		return false
	}

	if len(ks.keys) != len(b.keys) {
		return false
	}

	sort.Slice(ks.keys, func(i, j int) bool {
		return bytes.Compare(ks.keys[i].Key().Bytes(), ks.keys[j].Key().Bytes()) < 0
	})
	sort.Slice(b.keys, func(i, j int) bool {
		return bytes.Compare(b.keys[i].Key().Bytes(), b.keys[j].Key().Bytes()) < 0
	})

	for i := range ks.keys {
		if !ks.keys[i].Equal(b.keys[i]) {
			return false
		}
	}

	return true
}

func checkThreshold(fs []operation.FactSign, keys Keys) error {
	var sum uint
	for i := range fs {
		ky, found := keys.Key(fs[i].Signer())
		if !found {
			return errors.Errorf("unknown key found, %s", fs[i].Signer())
		}
		sum += ky.Weight()
	}

	if sum < keys.Threshold() {
		return errors.Errorf("not passed threshold, sum=%d < threshold=%d", sum, keys.Threshold())
	}

	return nil
}
