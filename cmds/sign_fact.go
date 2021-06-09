package cmds

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/operation"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/util"
)

type SignFactCommand struct {
	*BaseCommand
	Privatekey PrivatekeyFlag          `arg:"" name:"privatekey" help:"sender's privatekey" required:""`
	NetworkID  mitumcmds.NetworkIDFlag `name:"network-id" help:"network-id" required:""`
	Pretty     bool                    `name:"pretty" help:"pretty format"`
	Seal       FileLoad                `help:"seal" optional:""`
}

func NewSignFactCommand() SignFactCommand {
	return SignFactCommand{
		BaseCommand: NewBaseCommand("sign-fact"),
	}
}

func (cmd *SignFactCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return xerrors.Errorf("failed to initialize command: %w", err)
	}

	var sl operation.Seal
	if s, err := loadSeal(cmd.Seal.Bytes(), cmd.NetworkID.NetworkID()); err != nil {
		return err
	} else if so, ok := s.(operation.Seal); !ok {
		return xerrors.Errorf("seal is not operation.Seal, %T", s)
	} else if _, ok := so.(operation.SealUpdater); !ok {
		return xerrors.Errorf("seal is not operation.SealUpdater, %T", so)
	} else {
		sl = so
	}

	cmd.Log().Debug().Hinted("seal", sl.Hash()).Msg("seal loaded")

	nops := make([]operation.Operation, len(sl.Operations()))
	for i := range sl.Operations() {
		op := sl.Operations()[i]

		var fsu operation.FactSignUpdater
		if u, ok := op.(operation.FactSignUpdater); !ok {
			cmd.Log().Debug().
				Interface("operation", op).
				Str("operation_type", op.Hint().String()).
				Msg("not operation.FactSignUpdater")

			nops[i] = op
		} else {
			fsu = u
		}

		if sig, err := operation.NewFactSignature(cmd.Privatekey, op.Fact(), cmd.NetworkID.NetworkID()); err != nil {
			return err
		} else {
			f := operation.NewBaseFactSign(cmd.Privatekey.Publickey(), sig)

			if nop, err := fsu.AddFactSigns(f); err != nil {
				return err
			} else {
				nops[i] = nop.(operation.Operation)
			}
		}
	}

	sl = sl.(operation.SealUpdater).SetOperations(nops).(operation.Seal)

	if s, err := signSeal(sl, cmd.Privatekey, cmd.NetworkID.NetworkID()); err != nil {
		return err
	} else {
		sl = s.(operation.Seal)

		cmd.Log().Debug().Msg("seal signed")
	}

	cmd.pretty(cmd.Pretty, sl)

	return nil
}
