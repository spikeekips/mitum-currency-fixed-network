package cmds

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"golang.org/x/xerrors"

	mc "github.com/spikeekips/mitum-currency"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
)

type CreateAccountCommand struct {
	URL        *url.URL       `name:"node url" help:"remote mitum url (default: ${node_url})" required:"" default:"${node_url}"` // nolint
	Sender     AddressFlag    `arg:"" name:"sender" help:"sender address" required:""`
	Privatekey PrivatekeyFlag `arg:"" name:"privatekey" help:"sender's privatekey" required:""`
	Amount     AmountFlag     `arg:"" name:"amount" help:"amount to send" required:""`
	Threshold  uint           `help:"threshold for keys (default: ${create_account_threshold})" default:"${create_account_threshold}"` // nolint
	Token      string         `help:"token for operation" optional:""`
	NetworkID  string         `name:"network-id" help:"network-id" required:""`
	Keys       []KeyFlag      `name:"key" help:"key for new account (ex: \"<private key>,<weight>\")" sep:"@"`
	DryRun     bool           `help:"dry-run, print operation" optional:"" default:"false"`

	keys mc.Keys
}

func (cmd *CreateAccountCommand) Run(flags *MainFlags, version util.Version) error {
	var log logging.Logger
	if cmd.DryRun {
		log = logging.NilLogger
	} else if l, err := setupLogging(flags.LogFlags); err != nil {
		return err
	} else {
		log = l
	}

	log.Info().Str("version", version.String()).Msg("mitum-currency")
	log.Debug().Interface("flags", flags).Msg("flags parsed")
	defer log.Info().Msg("mitum-currency finished")

	if err := cmd.parseFlags(); err != nil {
		return err
	}

	var sl operation.Seal
	if s, err := cmd.createOperation(); err != nil {
		return err
	} else {
		sl = s
	}

	if cmd.DryRun {
		_, _ = fmt.Fprintln(os.Stdout, string(jsonenc.MustMarshalIndent(sl)))

		return nil
	}

	log.Debug().Hinted("seal", sl.Hash()).Msg("trying to send seal")

	if err := cmd.send(sl); err != nil {
		log.Error().Err(err).Msg("failed to send seal")

		return err
	}

	return nil
}

func (cmd *CreateAccountCommand) parseFlags() error {
	if len(cmd.Keys) < 1 {
		return xerrors.Errorf("--key must be given at least one")
	}

	if len(cmd.Token) < 1 {
		cmd.Token = localtime.String(localtime.Now())
	}

	{
		ks := make([]mc.Key, len(cmd.Keys))
		for i := range cmd.Keys {
			ks[i] = cmd.Keys[i].Key
		}

		if kys, err := mc.NewKeys(ks, cmd.Threshold); err != nil {
			return err
		} else if err := kys.IsValid(nil); err != nil {
			return err
		} else {
			cmd.keys = kys
		}
	}

	return nil
}

func (cmd *CreateAccountCommand) createOperation() (operation.Seal, error) {
	fact := mc.NewCreateAccountFact([]byte(cmd.Token), cmd.Sender.Address, cmd.keys, cmd.Amount.Amount)

	var fs []operation.FactSign
	if sig, err := operation.NewFactSignature(cmd.Privatekey, fact, []byte(cmd.NetworkID)); err != nil {
		return nil, err
	} else {
		fs = append(fs, operation.NewBaseFactSign(cmd.Privatekey.Publickey(), sig))
	}

	if op, err := mc.NewCreateAccount(fact, fs, ""); err != nil {
		return nil, xerrors.Errorf("failed to create create-account operation: %w", err)
	} else if sl, err := operation.NewBaseSeal(
		cmd.Privatekey,
		[]operation.Operation{op},
		[]byte(cmd.NetworkID),
	); err != nil {
		return nil, xerrors.Errorf("failed to create operation.Seal: %w", err)
	} else {
		return sl, nil
	}
}

func (cmd *CreateAccountCommand) send(sl operation.Seal) error {
	var channel network.NetworkChannel
	if ch, err := launcher.LoadNodeChannel(cmd.URL, encs); err != nil {
		return err
	} else {
		channel = ch
	}

	return channel.SendSeal(sl)
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
	if k, err := key.DecodePublickey(defaultJSONEnc, l[0]); err != nil {
		return xerrors.Errorf("invalid public key, %q for --key: %w", l[0], err)
	} else {
		pk = k
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
	} else {
		*v = AddressFlag{Address: a}
	}

	return nil
}

type PrivatekeyFlag struct {
	key.Privatekey
}

func (v *PrivatekeyFlag) UnmarshalText(b []byte) error {
	if k, err := key.DecodePrivatekey(defaultJSONEnc, string(b)); err != nil {
		return xerrors.Errorf("invalid private key, %q: %w", string(b), err)
	} else {
		*v = PrivatekeyFlag{Privatekey: k}
	}

	return nil
}

type AmountFlag struct {
	mc.Amount
}

func (v *AmountFlag) UnmarshalText(b []byte) error {
	if a, err := mc.NewAmountFromString(string(b)); err != nil {
		return xerrors.Errorf("invalid amount string, %q: %w", string(b), err)
	} else {
		*v = AmountFlag{Amount: a}
	}

	return nil
}
