package cmds

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/key"
	contestlib "github.com/spikeekips/mitum/contest/lib"

	"github.com/spikeekips/mitum-currency/currency"
)

type MainFlags struct {
	*contestlib.LogFlags
	Log     []string     `help:"log file"`
	Version struct{}     `cmd:"" help:"print version"` // TODO set ldflags
	Init    InitCommand  `cmd:"" help:"initialize"`
	Run     RunCommand   `cmd:"" help:"run node"`
	Node    NodeCommand  `cmd:"" name:"node" help:"various node commands"`
	Seal    SealCommand  `cmd:"" name:"seal" help:"generate seal"`
	Key     KeyCommand   `cmd:"" name:"key" help:"key"`
	Bench   BenchCommand `cmd:"" name:"bench" help:"benchmark"`
}

type KeyFlag struct {
	Key currency.Key
}

func (v *KeyFlag) UnmarshalText(b []byte) error {
	if bytes.Equal(bytes.TrimSpace(b), []byte("-")) {
		if c, err := loadFromStdInput(); err != nil {
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
	if k, err := key.DecodeKey(defaultJSONEnc, l[0]); err != nil {
		return xerrors.Errorf("invalid public key, %q for --key: %w", l[0], err)
	} else if priv, ok := k.(key.Privatekey); ok {
		pk = priv.Publickey()
	} else {
		pk = k.(key.Publickey)
	}

	var weight uint
	if i, err := strconv.ParseUint(l[1], 10, 64); err != nil {
		return xerrors.Errorf("invalid weight, %q for --key: %w", l[1], err)
	} else {
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

type AddressFlag struct {
	currency.Address
}

func (v *AddressFlag) UnmarshalText(b []byte) error {
	if a, err := currency.NewAddress(string(b)); err != nil {
		return xerrors.Errorf("invalid Address string, %q: %w", string(b), err)
	} else if err := a.IsValid(nil); err != nil {
		return err
	} else {
		*v = AddressFlag{Address: a}
	}

	return nil
}

type PrivatekeyFlag struct {
	key.Privatekey
	notEmpty bool
}

func (v PrivatekeyFlag) Empty() bool {
	return !v.notEmpty
}

func (v *PrivatekeyFlag) UnmarshalText(b []byte) error {
	if k, err := key.DecodePrivatekey(defaultJSONEnc, string(b)); err != nil {
		return xerrors.Errorf("invalid private key, %q: %w", string(b), err)
	} else if err := k.IsValid(nil); err != nil {
		return err
	} else {
		*v = PrivatekeyFlag{Privatekey: k}
	}

	v.notEmpty = true

	return nil
}

type AmountFlag struct {
	currency.Amount
}

func (v *AmountFlag) UnmarshalText(b []byte) error {
	if a, err := currency.NewAmountFromString(string(b)); err != nil {
		return xerrors.Errorf("invalid amount string, %q: %w", string(b), err)
	} else if err := a.IsValid(nil); err != nil {
		return err
	} else {
		*v = AmountFlag{Amount: a}
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
		if c, err := loadFromStdInput(); err != nil {
			return err
		} else {
			*v = c

			return nil
		}
	}

	if c, err := ioutil.ReadFile(filepath.Clean(string(b))); err != nil {
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

type StringLoad []byte

func (v *StringLoad) UnmarshalText(b []byte) error {
	if bytes.Equal(bytes.TrimSpace(b), []byte("-")) {
		if c, err := loadFromStdInput(); err != nil {
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
