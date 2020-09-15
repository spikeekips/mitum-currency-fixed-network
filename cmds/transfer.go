package cmds

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"

	currency "github.com/spikeekips/mitum-currency/currency"
)

type TransferCommand struct {
	BaseCommand
	Privatekey PrivatekeyFlag `arg:"" name:"privatekey" help:"sender's privatekey" required:""`
	Sender     AddressFlag    `arg:"" name:"sender" help:"sender address" required:""`
	Receiver   AddressFlag    `arg:"" name:"receiver" help:"receiver address" required:""`
	Amount     AmountFlag     `arg:"" name:"amount" help:"amount to send" required:""`
	Token      string         `help:"token for operation" optional:""`
	NetworkID  NetworkIDFlag  `name:"network-id" help:"network-id" required:""`
	Memo       string         `name:"memo" help:"memo"`
	Pretty     bool           `name:"pretty" help:"pretty format"`
	Seal       FileLoad       `help:"seal" optional:""`

	sender   base.Address
	receiver base.Address
}

func (cmd *TransferCommand) Run(flags *MainFlags, version util.Version, log logging.Logger) error {
	_ = cmd.BaseCommand.Run(flags, version, log)

	if err := cmd.parseFlags(); err != nil {
		return err
	} else if sender, err := cmd.Sender.Encode(defaultJSONEnc); err != nil {
		return xerrors.Errorf("invalid sender format, %q: %w", cmd.Sender.String(), err)
	} else if receiver, err := cmd.Receiver.Encode(defaultJSONEnc); err != nil {
		return xerrors.Errorf("invalid sender format, %q: %w", cmd.Sender.String(), err)
	} else {
		cmd.sender = sender
		cmd.receiver = receiver
	}

	var op operation.Operation
	if o, err := cmd.createOperation(); err != nil {
		return err
	} else {
		op = o
	}

	if sl, err := loadSealAndAddOperation(
		cmd.Seal.Bytes(),
		cmd.Privatekey,
		cmd.NetworkID.Bytes(),
		op,
	); err != nil {
		return err
	} else {
		cmd.pretty(cmd.Pretty, sl)
	}

	return nil
}

func (cmd *TransferCommand) parseFlags() error {
	if len(cmd.Token) < 1 {
		cmd.Token = localtime.String(localtime.Now())
	}

	return nil
}

func (cmd *TransferCommand) createOperation() (operation.Operation, error) {
	item := currency.NewTransferItem(cmd.receiver, cmd.Amount.Amount)
	if err := item.IsValid(nil); err != nil {
		return nil, err
	}

	items := []currency.TransferItem{item}
	fact := currency.NewTransfersFact([]byte(cmd.Token), cmd.sender, items)

	var fs []operation.FactSign
	if sig, err := operation.NewFactSignature(cmd.Privatekey, fact, cmd.NetworkID.Bytes()); err != nil {
		return nil, err
	} else {
		fs = append(fs, operation.NewBaseFactSign(cmd.Privatekey.Publickey(), sig))
	}

	if op, err := currency.NewTransfers(fact, fs, cmd.Memo); err != nil {
		return nil, xerrors.Errorf("failed to create create-account operation: %w", err)
	} else {
		return op, nil
	}
}
