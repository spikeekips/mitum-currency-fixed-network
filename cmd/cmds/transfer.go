package cmds

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util/localtime"

	mc "github.com/spikeekips/mitum-currency"
)

type TransferCommand struct {
	Privatekey PrivatekeyFlag `arg:"" name:"privatekey" help:"sender's privatekey" required:""`
	Sender     AddressFlag    `arg:"" name:"sender" help:"sender address" required:""`
	Receiver   AddressFlag    `arg:"" name:"receiver" help:"receiver address" required:""`
	Amount     AmountFlag     `arg:"" name:"amount" help:"amount to send" required:""`
	Token      string         `help:"token for operation" optional:""`
	NetworkID  string         `name:"network-id" help:"network-id" required:""`
	Memo       string         `name:"memo" help:"memo"`
	Pretty     bool           `name:"pretty" help:"pretty format"`
}

func (cmd *TransferCommand) Run() error {
	if err := cmd.parseFlags(); err != nil {
		return err
	}

	var sl operation.Seal
	if s, err := cmd.createSeal(); err != nil {
		return err
	} else {
		sl = s
	}

	prettyPrint(cmd.Pretty, sl)

	return nil
}

func (cmd *TransferCommand) parseFlags() error {
	if len(cmd.Token) < 1 {
		cmd.Token = localtime.String(localtime.Now())
	}

	if len(cmd.Memo) < 1 {
		if b, err := loadFromStdInput(); err != nil {
			return err
		} else {
			cmd.Memo = string(b)
		}
	}

	return nil
}

func (cmd *TransferCommand) createSeal() (operation.Seal, error) {
	fact := mc.NewTransferFact([]byte(cmd.Token), cmd.Sender.Address, cmd.Receiver.Address, cmd.Amount.Amount)

	var fs []operation.FactSign
	if sig, err := operation.NewFactSignature(cmd.Privatekey, fact, []byte(cmd.NetworkID)); err != nil {
		return nil, err
	} else {
		fs = append(fs, operation.NewBaseFactSign(cmd.Privatekey.Publickey(), sig))
	}

	if op, err := mc.NewTransfer(fact, fs, cmd.Memo); err != nil {
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
