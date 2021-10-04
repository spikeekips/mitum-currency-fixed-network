package cmds

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base/operation"
	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/util"
)

type SignFactCommand struct {
	*BaseCommand
	Privatekey PrivatekeyFlag          `arg:"" name:"privatekey" help:"sender's privatekey" required:"true"`
	NetworkID  mitumcmds.NetworkIDFlag `name:"network-id" help:"network-id" required:"true"`
	Pretty     bool                    `name:"pretty" help:"pretty format"`
	Seal       mitumcmds.FileLoad      `help:"seal" optional:""`
}

func NewSignFactCommand() SignFactCommand {
	return SignFactCommand{
		BaseCommand: NewBaseCommand("sign-fact"),
	}
}

func (cmd *SignFactCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return errors.Wrap(err, "failed to initialize command")
	}

	var sl operation.Seal
	if s, err := LoadSeal(cmd.Seal.Bytes(), cmd.NetworkID.NetworkID()); err != nil {
		return err
	} else if so, ok := s.(operation.Seal); !ok {
		return errors.Errorf("seal is not operation.Seal, %T", s)
	} else if _, ok := so.(operation.SealUpdater); !ok {
		return errors.Errorf("seal is not operation.SealUpdater, %T", so)
	} else {
		sl = so
	}

	cmd.Log().Debug().Stringer("seal", sl.Hash()).Msg("seal loaded")

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

		sig, err := operation.NewFactSignature(cmd.Privatekey, op.Fact(), cmd.NetworkID.NetworkID())
		if err != nil {
			return err
		}
		f := operation.NewBaseFactSign(cmd.Privatekey.Publickey(), sig)

		nop, err := fsu.AddFactSigns(f)
		if err != nil {
			return err
		}
		nops[i] = nop.(operation.Operation)
	}

	sl = sl.(operation.SealUpdater).SetOperations(nops).(operation.Seal)

	s, err := SignSeal(sl, cmd.Privatekey, cmd.NetworkID.NetworkID())
	if err != nil {
		return err
	}
	sl = s.(operation.Seal)

	cmd.Log().Debug().Msg("seal signed")

	PrettyPrint(cmd.Out, cmd.Pretty, sl)

	return nil
}
