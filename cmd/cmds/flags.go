package cmds

import (
	"strconv"
	"strings"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/key"
	contestlib "github.com/spikeekips/mitum/contest/lib"

	mc "github.com/spikeekips/mitum-currency"
)

type MainFlags struct {
	Version struct{} `cmd:"" help:"print version"` // TODO set ldflags
	*contestlib.LogFlags
	Init InitCommand `cmd:"" help:"initialize"`
	Run  RunCommand  `cmd:"" help:"run node"`
	Node NodeCommand `cmd:"" name:"node" help:"various node commands"`
	Send SendCommand `cmd:"" name:"send" help:"send seal to remote mitum node"`
	Seal SealCommand `cmd:"" name:"seal" help:"generate seal"`
	Key  KeyCommand  `cmd:"" name:"key" help:"key"`
}

type KeyFlag struct {
	Key mc.Key
}

func (v *KeyFlag) UnmarshalText(b []byte) error {
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

	k := mc.NewKey(pk, weight)
	if err := k.IsValid(nil); err != nil {
		return xerrors.Errorf("invalid key string: %w", err)
	} else {
		v.Key = k
	}

	return nil
}

type AddressFlag struct {
	mc.Address
}

func (v *AddressFlag) UnmarshalText(b []byte) error {
	if a, err := mc.NewAddress(string(b)); err != nil {
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
	mc.Amount
}

func (v *AmountFlag) UnmarshalText(b []byte) error {
	if a, err := mc.NewAmountFromString(string(b)); err != nil {
		return xerrors.Errorf("invalid amount string, %q: %w", string(b), err)
	} else if err := a.IsValid(nil); err != nil {
		return err
	} else {
		*v = AmountFlag{Amount: a}
	}

	return nil
}
