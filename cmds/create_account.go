package cmds

import (
	"bytes"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"

	"github.com/spikeekips/mitum-currency/currency"
)

type CreateAccountCommand struct {
	*BaseCommand
	OperationFlags
	Sender    AddressFlag    `arg:"" name:"sender" help:"sender address" required:"true"`
	Currency  CurrencyIDFlag `arg:"" name:"currency" help:"currency id" required:"true"`
	Big       BigFlag        `arg:"" name:"big" help:"big to send" required:"true"`
	Threshold uint           `help:"threshold for keys (default: ${create_account_threshold})" default:"${create_account_threshold}"` // nolint
	Keys      []KeyFlag      `name:"key" help:"key for new account (ex: \"<public key>,<weight>\")" sep:"@"`
	Seal      FileLoad       `help:"seal" optional:""`
	sender    base.Address
	keys      currency.Keys
}

func NewCreateAccountCommand() CreateAccountCommand {
	return CreateAccountCommand{
		BaseCommand: NewBaseCommand("create-account-operation"),
	}
}

func (cmd *CreateAccountCommand) Run(version util.Version) error { // nolint:dupl
	if err := cmd.Initialize(cmd, version); err != nil {
		return xerrors.Errorf("failed to initialize command: %w", err)
	}

	if err := cmd.parseFlags(); err != nil {
		return err
	}

	op, err := cmd.createOperation()
	if err != nil {
		return err
	}

	sl, err := loadSealAndAddOperation(
		cmd.Seal.Bytes(),
		cmd.Privatekey,
		cmd.NetworkID.NetworkID(),
		op,
	)
	if err != nil {
		return err
	}
	cmd.pretty(cmd.Pretty, sl)

	return nil
}

func (cmd *CreateAccountCommand) parseFlags() error {
	if err := cmd.OperationFlags.IsValid(nil); err != nil {
		return err
	}

	a, err := cmd.Sender.Encode(jenc)
	if err != nil {
		return xerrors.Errorf("invalid sender format, %q: %w", cmd.Sender.String(), err)
	}
	cmd.sender = a

	if len(cmd.Keys) < 1 {
		return xerrors.Errorf("--key must be given at least one")
	}

	{
		ks := make([]currency.Key, len(cmd.Keys))
		for i := range cmd.Keys {
			ks[i] = cmd.Keys[i].Key
		}

		if kys, err := currency.NewKeys(ks, cmd.Threshold); err != nil {
			return err
		} else if err := kys.IsValid(nil); err != nil {
			return err
		} else {
			cmd.keys = kys
		}
	}

	return nil
}

func (cmd *CreateAccountCommand) createOperation() (operation.Operation, error) { // nolint:dupl
	i, err := loadOperations(cmd.Seal.Bytes(), cmd.NetworkID.NetworkID())
	if err != nil {
		return nil, err
	}
	var items []currency.CreateAccountsItem
	for j := range i {
		if t, ok := i[j].(currency.CreateAccounts); ok {
			items = t.Fact().(currency.CreateAccountsFact).Items()
		}
	}

	am := currency.NewAmount(cmd.Big.Big, cmd.Currency.CID)
	if err = am.IsValid(nil); err != nil {
		return nil, err
	}

	item := currency.NewCreateAccountsItemSingleAmount(cmd.keys, am)
	if err = item.IsValid(nil); err != nil {
		return nil, err
	}
	items = append(items, item)

	fact := currency.NewCreateAccountsFact([]byte(cmd.Token), cmd.sender, items)

	sig, err := operation.NewFactSignature(cmd.Privatekey, fact, cmd.NetworkID.NetworkID())
	if err != nil {
		return nil, err
	}
	fs := []operation.FactSign{
		operation.NewBaseFactSign(cmd.Privatekey.Publickey(), sig),
	}

	op, err := currency.NewCreateAccounts(fact, fs, cmd.Memo)
	if err != nil {
		return nil, xerrors.Errorf("failed to create create-account operation: %w", err)
	}
	return op, nil
}

func loadSeal(b []byte, networkID base.NetworkID) (seal.Seal, error) {
	if len(bytes.TrimSpace(b)) < 1 {
		return nil, xerrors.Errorf("empty input")
	}

	if sl, err := seal.DecodeSeal(b, jenc); err != nil {
		return nil, err
	} else if err := sl.IsValid(networkID); err != nil {
		return nil, xerrors.Errorf("invalid seal: %w", err)
	} else {
		return sl, nil
	}
}

func loadSealAndAddOperation(
	b []byte,
	privatekey key.Privatekey,
	networkID base.NetworkID,
	op operation.Operation,
) (operation.Seal, error) {
	if b == nil {
		bs, err := operation.NewBaseSeal(
			privatekey,
			[]operation.Operation{op},
			networkID,
		)
		if err != nil {
			return nil, xerrors.Errorf("failed to create operation.Seal: %w", err)
		}
		return bs, nil
	}

	var sl operation.Seal
	if s, err := loadSeal(b, networkID); err != nil {
		return nil, err
	} else if so, ok := s.(operation.Seal); !ok {
		return nil, xerrors.Errorf("seal is not operation.Seal, %T", s)
	} else if _, ok := so.(operation.SealUpdater); !ok {
		return nil, xerrors.Errorf("seal is not operation.SealUpdater, %T", s)
	} else {
		sl = so
	}

	// NOTE add operation to existing seal
	sl = sl.(operation.SealUpdater).SetOperations([]operation.Operation{op}).(operation.Seal)

	s, err := signSeal(sl, privatekey, networkID)
	if err != nil {
		return nil, err
	}
	sl = s.(operation.Seal)

	return sl, nil
}
