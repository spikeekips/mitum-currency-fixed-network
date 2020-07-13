package cmds

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/seal"
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

func SetupLogging(flags *contestlib.LogFlags) (logging.Logger, error) {
	if o, err := contestlib.SetupLoggingOutput(flags.Log, flags.LogFormat, flags.LogColor, os.Stderr); err != nil {
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

func loadFromStdInput() ([]byte, error) {
	var b []byte
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			b = append(b, sc.Bytes()...)
		}

		if err := sc.Err(); err != nil {
			return nil, err
		}
	}

	return b, nil
}

func loadFromFileOrInput(f string) ([]byte, bool, error) {
	if len(f) > 0 {
		if b, err := ioutil.ReadFile(filepath.Clean(f)); err != nil {
			return nil, false, err
		} else {
			return b, true, nil
		}
	}

	if b, err := loadFromStdInput(); err != nil {
		return nil, false, err
	} else {
		return b, false, nil
	}
}

func signSeal(sl seal.Seal, priv key.Privatekey, networkID base.NetworkID) (seal.Seal, error) {
	p := reflect.New(reflect.TypeOf(sl))
	p.Elem().Set(reflect.ValueOf(sl))

	signer := p.Interface().(seal.Signer)

	if err := signer.Sign(priv, networkID); err != nil {
		return nil, err
	}

	return p.Elem().Interface().(seal.Seal), nil
}

func loadKeyFromFileOrInput(s string) (key.Key, bool, error) {
	var fromString bool
	if len(s) > 0 {
		fromString = true
	} else if b, err := loadFromStdInput(); err != nil {
		return nil, false, err
	} else {
		s = string(b)
	}

	s = strings.TrimSpace(s)

	if pk, err := key.DecodeKey(defaultJSONEnc, s); err != nil {
		return nil, fromString, err
	} else if err := pk.IsValid(nil); err != nil {
		return nil, fromString, err
	} else {
		return pk, fromString, nil
	}
}
