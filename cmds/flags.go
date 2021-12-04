package cmds

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"

	"github.com/spikeekips/mitum-currency/currency"
)

type KeyFlag struct {
	Key currency.BaseAccountKey
}

func (v *KeyFlag) UnmarshalText(b []byte) error {
	if bytes.Equal(bytes.TrimSpace(b), []byte("-")) {
		c, err := mitumcmds.LoadFromStdInput()
		if err != nil {
			return err
		}
		b = c
	}

	l := strings.SplitN(string(b), ",", 2)
	if len(l) != 2 {
		return errors.Errorf(`wrong formatted; "<string private key>,<uint weight>"`)
	}

	var pk key.Publickey
	if k, err := key.DecodeKey(jenc, l[0]); err != nil {
		return errors.Wrapf(err, "invalid public key, %q for --key", l[0])
	} else if priv, ok := k.(key.Privatekey); ok {
		pk = priv.Publickey()
	} else {
		pk = k.(key.Publickey)
	}

	var weight uint = 100
	if i, err := strconv.ParseUint(l[1], 10, 8); err != nil {
		return errors.Wrapf(err, "invalid weight, %q for --key", l[1])
	} else if i > 0 && i <= 100 {
		weight = uint(i)
	}

	if k, err := currency.NewBaseAccountKey(pk, weight); err != nil {
		return err
	} else if err := k.IsValid(nil); err != nil {
		return errors.Wrap(err, "invalid key string")
	} else {
		v.Key = k
	}

	return nil
}

type StringLoad []byte

func (v *StringLoad) UnmarshalText(b []byte) error {
	if bytes.Equal(bytes.TrimSpace(b), []byte("-")) {
		c, err := mitumcmds.LoadFromStdInput()
		if err != nil {
			return err
		}
		*v = c

		return nil
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
		return errors.Wrapf(err, "invalid private key, %q", string(b))
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
	hs, err := hint.ParseHintedString(string(b))
	if err != nil {
		return err
	}
	v.s = string(b)
	v.ad = base.AddressDecoder{HintedString: encoder.NewHintedString(hs.Hint(), hs.Body())}

	return nil
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
		return errors.Wrapf(err, "invalid big string, %q", string(b))
	} else if err := a.IsValid(nil); err != nil {
		return err
	} else {
		*v = BigFlag{Big: a}
	}

	return nil
}

type CurrencyIDFlag struct {
	CID currency.CurrencyID
}

func (v *CurrencyIDFlag) UnmarshalText(b []byte) error {
	cid := currency.CurrencyID(string(b))
	if err := cid.IsValid(nil); err != nil {
		return err
	}
	v.CID = cid

	return nil
}

func (v *CurrencyIDFlag) String() string {
	return v.CID.String()
}

type CurrencyAmountFlag struct {
	CID currency.CurrencyID
	Big currency.Big
}

func (v *CurrencyAmountFlag) UnmarshalText(b []byte) error {
	l := strings.SplitN(string(b), ",", 2)
	if len(l) != 2 {
		return fmt.Errorf("invalid currency-amount, %q", string(b))
	}

	a, c := l[0], l[1]

	cid := currency.CurrencyID(a)
	if err := cid.IsValid(nil); err != nil {
		return err
	}
	v.CID = cid

	if a, err := currency.NewBigFromString(c); err != nil {
		return errors.Wrapf(err, "invalid big string, %q", string(b))
	} else if err := a.IsValid(nil); err != nil {
		return err
	} else {
		v.Big = a
	}

	return nil
}

func (v *CurrencyAmountFlag) String() string {
	return v.CID.String() + "," + v.Big.String()
}
