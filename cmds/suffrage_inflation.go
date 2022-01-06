package cmds

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/isvalid"
)

type SuffrageInflationItemFlag struct {
	s        string
	receiver base.Address
	amount   currency.Amount
}

func (v *SuffrageInflationItemFlag) String() string {
	return v.s
}

func (v *SuffrageInflationItemFlag) UnmarshalText(b []byte) error {
	v.s = string(b)

	l := strings.SplitN(string(b), ",", 3)
	if len(l) != 3 {
		return isvalid.InvalidError.Errorf("invalid inflation amount, %q", string(b))
	}

	a, c := l[0], l[1]+","+l[2]

	af := &AddressFlag{}
	if err := af.UnmarshalText([]byte(a)); err != nil {
		return isvalid.InvalidError.Errorf("invalid inflation receiver address: %w", err)
	}

	receiver, err := af.Encode(jenc)
	if err != nil {
		return isvalid.InvalidError.Errorf("invalid inflation receiver address: %w", err)
	}

	v.receiver = receiver

	cf := &CurrencyAmountFlag{}
	if err := cf.UnmarshalText([]byte(c)); err != nil {
		return isvalid.InvalidError.Errorf("invalid inflation amount: %w", err)
	}
	v.amount = currency.NewAmount(cf.Big, cf.CID)

	return nil
}

func (v *SuffrageInflationItemFlag) IsValid([]byte) error {
	if err := isvalid.Check(nil, false, v.receiver, v.amount); err != nil {
		return err
	}

	if !v.amount.Big().OverZero() {
		return isvalid.InvalidError.Errorf("amount should be over zero")
	}

	return nil
}

type SuffrageInflationCommand struct {
	*BaseCommand
	OperationFlags
	Items []SuffrageInflationItemFlag `arg:"" name:"inflation item" help:"ex: \"<receiver address>,<currency>,<amount>\""`
	items []currency.SuffrageInflationItem
}

func NewSuffrageInflationCommand() SuffrageInflationCommand {
	return SuffrageInflationCommand{
		BaseCommand: NewBaseCommand("suffrage-inflation-operation"),
	}
}

func (cmd *SuffrageInflationCommand) Run(version util.Version) error { // nolint:dupl
	if err := cmd.Initialize(cmd, version); err != nil {
		return errors.Wrap(err, "failed to initialize command")
	}

	if err := cmd.parseFlags(); err != nil {
		return err
	}

	var op operation.Operation
	if i, err := cmd.createOperation(); err != nil {
		return errors.Wrap(err, "failed to create suffrage inflation operation")
	} else if err := i.IsValid([]byte(cmd.OperationFlags.NetworkID)); err != nil {
		return errors.Wrap(err, "invalid suffrage inflation operation")
	} else {
		cmd.Log().Debug().Interface("operation", i).Msg("operation loaded")

		op = i
	}

	i, err := operation.NewBaseSeal(
		cmd.OperationFlags.Privatekey,
		[]operation.Operation{op},
		[]byte(cmd.OperationFlags.NetworkID),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create operation.Seal")
	}
	cmd.Log().Debug().Interface("seal", i).Msg("seal loaded")

	PrettyPrint(cmd.Out, cmd.Pretty, i)

	return nil
}

func (cmd *SuffrageInflationCommand) parseFlags() error {
	if err := cmd.OperationFlags.IsValid(nil); err != nil {
		return err
	}

	if len(cmd.Items) < 1 {
		return fmt.Errorf("empty item flags")
	}

	items := make([]currency.SuffrageInflationItem, len(cmd.Items))
	for i := range cmd.Items {
		item := cmd.Items[i]
		if err := item.IsValid(nil); err != nil {
			return err
		}

		items[i] = currency.NewSuffrageInflationItem(item.receiver, item.amount)

		cmd.Log().Debug().
			Stringer("amount", item.amount).
			Stringer("receiver", item.receiver).
			Msg("inflation item loaded")
	}

	cmd.items = items

	return nil
}

func (cmd *SuffrageInflationCommand) createOperation() (currency.SuffrageInflation, error) {
	fact := currency.NewSuffrageInflationFact([]byte(cmd.Token), cmd.items)

	var fs []base.FactSign
	sig, err := base.NewFactSignature(
		cmd.OperationFlags.Privatekey,
		fact,
		[]byte(cmd.OperationFlags.NetworkID),
	)
	if err != nil {
		return currency.SuffrageInflation{}, err
	}
	fs = append(fs, base.NewBaseFactSign(cmd.OperationFlags.Privatekey.Publickey(), sig))

	return currency.NewSuffrageInflation(fact, fs, cmd.OperationFlags.Memo)
}
