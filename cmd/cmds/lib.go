package cmds

import (
	"golang.org/x/xerrors"

	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"

	mc "github.com/spikeekips/mitum-currency"
)

var (
	hinters        []hint.Hinter
	encs           *encoder.Encoders
	defaultJSONEnc *jsonenc.Encoder
)

func init() {
	hinters = append(hinters, contestlib.Hinters...)
	hinters = append(hinters,
		mc.Address(""),
		mc.Key{},
		mc.Keys{},
		mc.GenesisAccount{},
		mc.GenesisAccountFact{},
		mc.CreateAccount{},
		mc.CreateAccountFact{},
		mc.Transfer{},
		mc.TransferFact{},
	)

	if es, err := loadEncoders(); err != nil {
		panic(err)
	} else if e, err := es.Encoder(jsonenc.JSONType, ""); err != nil {
		panic(err)
	} else {
		encs = es
		defaultJSONEnc = e.(*jsonenc.Encoder)
	}
}

type MainFlags struct {
	Version struct{} `cmd:"" help:"print version"` // TODO set ldflags
	*contestlib.LogFlags
	Init InitCommand `cmd:"" help:"initialize"`
	Run  RunCommand  `cmd:"" help:"run node"`
	Node NodeCommand `cmd:"" name:"node" help:"various node commands"`
	Send SendCommand `cmd:"" name:"send" help:"send seal to remote mitum node"`
}

func setupLogging(flags *contestlib.LogFlags) (logging.Logger, error) {
	if o, err := contestlib.SetupLoggingOutput(flags.Log, flags.LogFormat, flags.LogColor); err != nil {
		return logging.NilLogger, err
	} else if l, err := contestlib.SetupLogging(o, flags.LogLevel.Zero(), flags.Verbose); err != nil {
		return logging.NilLogger, err
	} else {
		return l, nil
	}
}

func loadEncoders() (*encoder.Encoders, error) {
	if e, err := encoder.LoadEncoders(
		[]encoder.Encoder{jsonenc.NewEncoder(), bsonenc.NewEncoder()},
		hinters...,
	); err != nil {
		return nil, xerrors.Errorf("failed to load encoders: %w", err)
	} else {
		return e, nil
	}
}

func createLauncherFromDesign(f string, version util.Version, log logging.Logger) (*mc.Launcher, error) {
	var encs *encoder.Encoders
	if e, err := loadEncoders(); err != nil {
		return nil, err
	} else {
		encs = e
	}

	var design *mc.NodeDesign
	if d, err := mc.LoadDesign(f, encs); err != nil {
		return nil, xerrors.Errorf("failed to load design: %w", err)
	} else {
		design = d
	}

	var nr *mc.Launcher
	if n, err := mc.NewLauncherFromDesign(design, version); err != nil {
		return nil, xerrors.Errorf("failed to create node runner: %w", err)
	} else if err := n.AddHinters(hinters...); err != nil {
		return nil, err
	} else {
		nr = n
	}

	log.Debug().Interface("design", design).Msg("load launcher from design")
	_ = nr.SetLogger(log)

	return nr, nil
}
