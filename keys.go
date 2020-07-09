package mc

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	KeyType  = hint.MustNewType(0xa0, 0x03, "mitum-currency-key")
	KeyHint  = hint.MustHint(KeyType, "0.0.1")
	KeysType = hint.MustNewType(0xa0, 0x04, "mitum-currency-keys")
	KeysHint = hint.MustHint(KeysType, "0.0.1")
)

type Key struct {
	k key.Publickey
	w uint
}

func NewKey(k key.Publickey, w uint) Key {
	return Key{k: k, w: w}
}

func (ky Key) IsValid([]byte) error {
	if ky.w < 1 || ky.w > 100 {
		return xerrors.Errorf("invalid key weight, 1 <= weight <= 100")
	}

	if err := ky.k.IsValid(nil); err != nil {
		return err
	}

	return nil
}

func (ky Key) Weight() uint {
	return ky.w
}

func (ky Key) Key() key.Publickey {
	return ky.k
}

func (ky Key) Hint() hint.Hint {
	return KeyHint
}

func (ky Key) Bytes() []byte {
	return util.ConcatBytesSlice([]byte(ky.k.String()), util.UintToBytes(ky.w))
}

type Keys struct {
	h         valuehash.Hash
	keys      []Key
	threshold uint
}

func NewKeys(keys []Key, threshold uint) (Keys, error) {
	ks := Keys{keys: keys, threshold: threshold}
	if h, err := ks.GenerateHash(); err != nil {
		return Keys{}, err
	} else {
		ks.h = h
	}

	return ks, nil
}

func (ks Keys) Hint() hint.Hint {
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
	for i := range ks.keys {
		bs[i] = ks.keys[i].Bytes()
	}

	bs[len(ks.keys)] = util.UintToBytes(ks.threshold)

	return util.ConcatBytesSlice(bs...)
}

func (ks Keys) IsValid([]byte) error {
	if ks.threshold < 1 || ks.threshold > 100 {
		return xerrors.Errorf("invalid threshold, %d, should be 1 <= threshold <= 100", ks.threshold)
	}

	if err := ks.h.IsValid(nil); err != nil {
		return err
	}

	if n := len(ks.keys); n < 1 {
		return xerrors.Errorf("empty keys")
	} else if n > 100 {
		return xerrors.Errorf("keys over 100, %d", n)
	}

	m := map[string]struct{}{}
	for i := range ks.keys {
		key := ks.keys[i]
		if err := key.IsValid(nil); err != nil {
			return err
		} else if _, found := m[key.Key().String()]; found {
			return xerrors.Errorf("duplicated keys found")
		} else {
			m[key.Key().String()] = struct{}{}
		}
	}

	var totalWeight uint
	for i := range ks.keys {
		totalWeight += ks.keys[i].Weight()
	}

	if totalWeight > 100 {
		return xerrors.Errorf("sum of weight of keys over 100, %d", totalWeight)
	}

	if h, err := ks.GenerateHash(); err != nil {
		return err
	} else if !ks.h.Equal(h) {
		return xerrors.Errorf("has not matched")
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

func checkThreshold(fs []operation.FactSign, keys Keys) error {
	var sum uint
	for i := range fs {
		if ky, found := keys.Key(fs[i].Signer()); found {
			sum += ky.Weight()
		} else {
			return xerrors.Errorf("unknown key found, %s", fs[i].Signer())
		}
	}

	if sum < keys.Threshold() {
		return xerrors.Errorf("not passed threshold, sum=%d < threshold=%d", sum, keys.Threshold())
	}

	return nil
}
