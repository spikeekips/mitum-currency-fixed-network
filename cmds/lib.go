package cmds

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"

	"github.com/spikeekips/mitum-currency/currency"
)

var (
	hinters        []hint.Hinter
	encs           *encoder.Encoders
	defaultJSONEnc *jsonenc.Encoder
)

func init() {
	hinters = append(hinters, contestlib.Hinters...)
	hinters = append(hinters,
		NodeInfo{},
		currency.Address(""),
		currency.Key{},
		currency.Keys{},
		currency.GenesisAccount{},
		currency.GenesisAccountFact{},
		currency.CreateAccounts{},
		currency.CreateAccountsFact{},
		currency.Transfers{},
		currency.TransfersFact{},
		currency.KeyUpdater{},
		currency.KeyUpdaterFact{},
		currency.AmountState{},
		currency.FeeOperationFact{},
		currency.FeeOperation{},
		currency.Account{},
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

func SetupLogging(logs []string, flags *contestlib.LogFlags) (logging.Logger, error) {
	if len(logs) < 1 {
		logs = append(logs, "/dev/stderr")
	}

	var outputs []io.Writer
	for _, l := range logs {
		if o, err := contestlib.SetupLoggingOutput(l, flags.LogFormat, flags.LogColor, os.Stderr); err != nil {
			return logging.Logger{}, err
		} else {
			outputs = append(outputs, o)
		}
	}

	var output io.Writer
	switch n := len(outputs); {
	case n == 1:
		output = outputs[0]
	default:
		output = zerolog.MultiLevelWriter(outputs...)
	}

	if l, err := contestlib.SetupLogging(output, flags.LogLevel.Zero(), flags.Verbose); err != nil {
		return logging.NilLogger, err
	} else {
		return l, nil
	}
}

type printCommand struct {
	o io.Writer
}

func (cm *printCommand) out() io.Writer {
	if cm.o != nil {
		return cm.o
	} else {
		return os.Stdout
	}
}

func (cm *printCommand) print(f string, a ...interface{}) {
	_, _ = fmt.Fprintf(cm.out(), f, a...)
	_, _ = fmt.Fprintln(cm.out())
}

func (cm *printCommand) pretty(pretty bool, i interface{}) {
	var b []byte
	if pretty {
		b = jsonenc.MustMarshalIndent(i)
	} else {
		b = jsonenc.MustMarshal(i)
	}

	_, _ = fmt.Fprintln(cm.out(), string(b))
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

func createLauncherFromDesign(b []byte, version util.Version, log logging.Logger) (*Launcher, error) {
	var design *NodeDesign
	if d, err := LoadNodeDesign(b, encs); err != nil {
		return nil, err
	} else {
		design = d
	}

	var nr *Launcher
	if n, err := NewLauncherFromDesign(design, version); err != nil {
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

func signSeal(sl seal.Seal, priv key.Privatekey, networkID base.NetworkID) (seal.Seal, error) {
	p := reflect.New(reflect.TypeOf(sl))
	p.Elem().Set(reflect.ValueOf(sl))

	signer := p.Interface().(seal.Signer)

	if err := signer.Sign(priv, networkID); err != nil {
		return nil, err
	}

	return p.Elem().Interface().(seal.Seal), nil
}

func loadFromStdInput() ([]byte, error) {
	var b []byte
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			b = append(b, sc.Bytes()...)
			b = append(b, []byte("\n")...)
		}

		if err := sc.Err(); err != nil {
			return nil, err
		}
	}

	return bytes.TrimSpace(b), nil
}

func loadSeal(b []byte, networkID base.NetworkID) (seal.Seal, error) {
	if len(bytes.TrimSpace(b)) < 1 {
		return nil, xerrors.Errorf("empty input")
	}

	if sl, err := seal.DecodeSeal(defaultJSONEnc, b); err != nil {
		return nil, err
	} else if err := sl.IsValid(networkID); err != nil {
		return nil, xerrors.Errorf("invalid seal: %w", err)
	} else {
		return sl, nil
	}
}

func loadKey(b []byte) (key.Key, error) {
	s := strings.TrimSpace(string(b))

	if pk, err := key.DecodeKey(defaultJSONEnc, s); err != nil {
		return nil, err
	} else if err := pk.IsValid(nil); err != nil {
		return nil, err
	} else {
		return pk, nil
	}
}

func prettyPrint(pretty bool, i interface{}) {
	var b []byte
	if pretty {
		b = jsonenc.MustMarshalIndent(i)
	} else {
		b = jsonenc.MustMarshal(i)
	}

	_, _ = fmt.Fprintln(os.Stdout, string(b))
}

func loadSealAndAddOperation(
	b []byte,
	privatekey key.Privatekey,
	networkID base.NetworkID,
	op operation.Operation,
) (operation.Seal, error) {
	if b == nil {
		if bs, err := operation.NewBaseSeal(
			privatekey,
			[]operation.Operation{op},
			networkID,
		); err != nil {
			return nil, xerrors.Errorf("failed to create operation.Seal: %w", err)
		} else {
			return bs, nil
		}
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
	sl = sl.(operation.SealUpdater).SetOperations(
		append(sl.Operations(), op),
	).(operation.Seal)

	if s, err := signSeal(sl, privatekey, networkID); err != nil {
		return nil, err
	} else {
		sl = s.(operation.Seal)
	}

	return sl, nil
}

func initlaizeProposalProcessor(dp isaac.ProposalProcessor, opr isaac.OperationProcessor) error {
	if _, err := dp.AddOperationProcessor(currency.Transfers{}, opr); err != nil {
		return err
	} else if _, err := dp.AddOperationProcessor(currency.CreateAccounts{}, opr); err != nil {
		return err
	} else if _, err := dp.AddOperationProcessor(currency.KeyUpdater{}, opr); err != nil {
		return err
	} else {
		return nil
	}
}

func saveGenesisAccountInfo(st storage.Storage, genesisBlock block.Block) (currency.Account, currency.Amount, error) {
	log.Debug().Msg("trying to save genesis info")
	var gac currency.Account
	var gbalance currency.Amount = currency.NilAmount
	for i := range genesisBlock.States() {
		st := genesisBlock.States()[i]
		if currency.IsStateAccountKey(st.Key()) {
			if ac, err := currency.LoadStateAccountValue(st); err != nil {
				return currency.Account{}, currency.NilAmount, err
			} else {
				gac = ac
			}
		} else if currency.IsStateBalanceKey(st.Key()) {
			if am, err := currency.StateAmountValue(st); err != nil {
				return currency.Account{}, currency.NilAmount, err
			} else {
				gbalance = am
			}
		}
	}

	if gac.IsEmpty() {
		return currency.Account{}, currency.NilAmount, xerrors.Errorf("failed to find genesis account")
	}

	if gbalance.Compare(currency.ZeroAmount) <= 0 {
		return currency.Account{}, currency.NilAmount, xerrors.Errorf("failed to find genesis balance")
	}

	if b, err := defaultJSONEnc.Marshal(gac); err != nil {
		return currency.Account{}, currency.NilAmount, xerrors.Errorf("failed to save genesis account: %w", err)
	} else if err := st.SetInfo(GenesisAccountKey, b); err != nil {
		return currency.Account{}, currency.NilAmount, xerrors.Errorf("failed to save genesis account: %w", err)
	}

	if b, err := defaultJSONEnc.Marshal(gbalance); err != nil {
		return currency.Account{}, currency.NilAmount, xerrors.Errorf("failed to save genesis balance: %w", err)
	} else if err := st.SetInfo(GenesisBalanceKey, b); err != nil {
		return currency.Account{}, currency.NilAmount, xerrors.Errorf("failed to save genesis balance: %w", err)
	}

	log.Debug().Msg("genesis info saved")
	return gac, gbalance, nil
}

func loadGenesisAccountInfo(st storage.Storage) (currency.Account, currency.Amount, error) {
	log.Debug().Msg("tryingo to load genesis info")
	var ac currency.Account
	var balance currency.Amount

	switch b, found, err := st.GetInfo(GenesisAccountKey); {
	case err != nil:
		return currency.Account{}, currency.NilAmount, xerrors.Errorf("failed to get genesis account: %w", err)
	case !found:
		return currency.Account{}, currency.NilAmount, storage.NotFoundError.Errorf("genesis account not found")
	default:
		if err := defaultJSONEnc.Decode(b, &ac); err != nil {
			return currency.Account{},
				currency.NilAmount,
				xerrors.Errorf("failed to load genesis account for getting fee receiver: %w", err)
		}
	}

	switch b, found, err := st.GetInfo(GenesisBalanceKey); {
	case err != nil:
		return currency.Account{}, currency.NilAmount, xerrors.Errorf("failed to get genesis balance: %w", err)
	case !found:
		return currency.Account{}, currency.NilAmount, storage.NotFoundError.Errorf("genesis balance not found")
	default:
		if err := defaultJSONEnc.Decode(b, &balance); err != nil {
			return currency.Account{}, currency.NilAmount, xerrors.Errorf("failed to load genesis balance: %w", err)
		}
	}

	log.Debug().Msg("genesis info loaded")
	return ac, balance, nil
}
