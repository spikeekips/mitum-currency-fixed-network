package cmds

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"

	"github.com/spikeekips/mitum-currency/currency"
)

type KeyFlag struct {
	Key currency.Key
}

func (v *KeyFlag) UnmarshalText(b []byte) error {
	if bytes.Equal(bytes.TrimSpace(b), []byte("-")) {
		if c, err := mitumcmds.LoadFromStdInput(); err != nil {
			return err
		} else {
			b = c
		}
	}

	l := strings.SplitN(string(b), ",", 2)
	if len(l) != 2 {
		return xerrors.Errorf(`wrong formatted; "<string private key>,<uint weight>"`)
	}

	var pk key.Publickey
	if k, err := key.DecodeKey(jenc, l[0]); err != nil {
		return xerrors.Errorf("invalid public key, %q for --key: %w", l[0], err)
	} else if priv, ok := k.(key.Privatekey); ok {
		pk = priv.Publickey()
	} else {
		pk = k.(key.Publickey)
	}

	var weight uint = 100
	if i, err := strconv.ParseUint(l[1], 10, 8); err != nil {
		return xerrors.Errorf("invalid weight, %q for --key: %w", l[1], err)
	} else if i > 0 && i <= 100 {
		weight = uint(i)
	}

	if k, err := currency.NewKey(pk, weight); err != nil {
		return err
	} else if err := k.IsValid(nil); err != nil {
		return xerrors.Errorf("invalid key string: %w", err)
	} else {
		v.Key = k
	}

	return nil
}

type StringLoad []byte

func (v *StringLoad) UnmarshalText(b []byte) error {
	if bytes.Equal(bytes.TrimSpace(b), []byte("-")) {
		if c, err := mitumcmds.LoadFromStdInput(); err != nil {
			return err
		} else {
			*v = c

			return nil
		}
	}

	*v = b

	return nil
}

func (v StringLoad) Bytes() []byte {
	return []byte(v)
}

func (v StringLoad) String() string {
	return string(v)
}

type PrivatekeyFlag struct {
	key.Privatekey
	notEmpty bool
}

func (v PrivatekeyFlag) Empty() bool {
	return !v.notEmpty
}

func (v *PrivatekeyFlag) UnmarshalText(b []byte) error {
	if k, err := key.DecodePrivatekey(jenc, string(b)); err != nil {
		return xerrors.Errorf("invalid private key, %q: %w", string(b), err)
	} else if err := k.IsValid(nil); err != nil {
		return err
	} else {
		*v = PrivatekeyFlag{Privatekey: k}
	}

	v.notEmpty = true

	return nil
}

type AddressFlag struct {
	s  string
	ad base.AddressDecoder
}

func (v *AddressFlag) UnmarshalText(b []byte) error {
	if ht, s, err := hint.ParseHintedString(string(b)); err != nil {
		return err
	} else {
		v.s = string(b)
		v.ad = base.AddressDecoder{HintedString: encoder.NewHintedString(ht, s)}

		return nil
	}
}

func (v *AddressFlag) String() string {
	return v.s
}

func (v *AddressFlag) Encode(enc encoder.Encoder) (base.Address, error) {
	return v.ad.Encode(enc)
}

type BigFlag struct {
	currency.Big
}

func (v *BigFlag) UnmarshalText(b []byte) error {
	if a, err := currency.NewBigFromString(string(b)); err != nil {
		return xerrors.Errorf("invalid big string, %q: %w", string(b), err)
	} else if err := a.IsValid(nil); err != nil {
		return err
	} else {
		*v = BigFlag{Big: a}
	}

	return nil
}

type NetworkIDFlag []byte

func (v *NetworkIDFlag) UnmarshalText(b []byte) error {
	*v = b

	return nil
}

func (v NetworkIDFlag) Bytes() []byte {
	return []byte(v)
}

type FileLoad []byte

func (v *FileLoad) UnmarshalText(b []byte) error {
	if bytes.Equal(bytes.TrimSpace(b), []byte("-")) {
		if c, err := mitumcmds.LoadFromStdInput(); err != nil {
			return err
		} else {
			*v = c

			return nil
		}
	}

	if c, err := os.ReadFile(filepath.Clean(string(b))); err != nil {
		return err
	} else {
		*v = c

		return nil
	}
}

func (v FileLoad) Bytes() []byte {
	return []byte(v)
}

func (v FileLoad) String() string {
	return string(v)
}

type CurrencyIDFlag struct {
	CID currency.CurrencyID
}

func (v *CurrencyIDFlag) UnmarshalText(b []byte) error {
	cid := currency.CurrencyID(string(b))
	if err := cid.IsValid(nil); err != nil {
		return err
	} else {
		v.CID = cid

		return nil
	}
}

func (v *CurrencyIDFlag) String() string {
	return v.CID.String()
}
