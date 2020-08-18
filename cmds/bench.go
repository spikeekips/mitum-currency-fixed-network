package cmds

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/storage"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum-currency/currency"
)

var genesisAmountString string = "99999999999999999999999999"

type BenchCommand struct {
	StorageURI *url.URL `name:"storage" help:"mongodb storage uri (default:mongodb://localhost:27017)" default:"mongodb://localhost:27017"` // nolint
	Operations uint     `arg:"" name:"operations" help:"number of operations (default:10)" default:"10"`
	NoHeader   bool     `name:"no-header" help:"don't print header output" default:"false"`
	log        logging.Logger
	ops        []operation.Operation
	local      *isaac.Localstate
	senderPriv key.Privatekey
	sender     currency.Address
	storage    storage.Storage
	suffrage   base.Suffrage
	networkID  base.NetworkID
	dp         isaac.ProposalProcessor
}

func (cmd *BenchCommand) Run(flags *MainFlags, version util.Version) error {
	var log logging.Logger
	if l, err := SetupLogging(flags.Log, flags.LogFlags); err != nil {
		return err
	} else {
		log = l
	}

	if cmd.Operations < 1 {
		return xerrors.Errorf("operations should be over 0")
	}

	log.Info().Str("version", version.String()).Msg("mitum-currency")
	log.Debug().Interface("flags", flags).Msg("flags parsed")
	defer log.Info().Msg("mitum-currency finished")

	log.Info().Msg("trying to benchmark")

	cmd.log = log

	return cmd.run()
}

func (cmd *BenchCommand) run() error {
	if !cmd.NoHeader {
		_, _ = fmt.Fprintln(os.Stdout, `date,ops,type,value`)
	}

	cmd.networkID = util.UUID().Bytes()
	cmd.senderPriv = key.MustNewBTCPrivatekey()

	if st, err := cmd.prepareStorage(); err != nil {
		return err
	} else {
		defer func() {
			if m, ok := st.(*mongodbstorage.Storage); ok {
				_ = m.Client().DropDatabase()
			}
		}()

		cmd.storage = st
	}
	cmd.log.Debug().Msg("storage prepared")

	if local, err := cmd.localstate(); err != nil {
		return err
	} else {
		cmd.local = local
	}
	cmd.log.Debug().Msg("local prepared")

	cmd.suffrage = base.NewFixedSuffrage(cmd.local.Node().Address(), []base.Address{cmd.local.Node().Address()})
	if err := cmd.suffrage.Initialize(); err != nil {
		return err
	}
	cmd.log.Debug().Msg("fixed suffrage prepared")

	cmd.log.Debug().Uint("operations", cmd.Operations).Msg("trying to create operations")
	if ops, err := cmd.newOperations(cmd.Operations); err != nil {
		return err
	} else {
		cmd.ops = ops
	}

	if err := cmd.bench(); err != nil {
		return err
	}

	return nil
}

func (cmd *BenchCommand) bench() error {
	cmd.log.Debug().Msg("trying to bench")

	cmd.dp = isaac.NewDefaultProposalProcessor(cmd.local, cmd.suffrage)
	cmd.dp.(logging.SetLogger).SetLogger(cmd.log)

	if _, err := cmd.dp.AddOperationProcessor(currency.Transfer{}, &currency.OperationProcessor{}); err != nil {
		return err
	}
	if _, err := cmd.dp.AddOperationProcessor(currency.CreateAccount{}, &currency.OperationProcessor{}); err != nil {
		return err
	}

	var proposal ballot.Proposal
	var ivp base.Voteproof

	cmd.log.Debug().Msg("trying to prepare")
	s := time.Now()
	if p, v, err := cmd.prepare(); err != nil {
		return err
	} else {
		elapsed := time.Since(s)
		cmd.log.Debug().Dur("elapsed", elapsed).Msg("prepared")
		printCSV(cmd.Operations, "bench-prepare", elapsed)

		proposal = p
		ivp = v
	}

	var blk block.Block
	var bs storage.BlockStorage

	s = time.Now()
	if a, b, err := cmd.process(proposal, ivp); err != nil {
		return err
	} else {
		elapsed := time.Since(s)
		cmd.log.Debug().Dur("elapsed", elapsed).Msg("processed")

		printCSV(cmd.Operations, "bench-process", elapsed)

		for k, v := range cmd.dp.States() {
			printCSV(cmd.Operations, "pp-"+k, v)
		}

		blk = a
		bs = b
	}

	s = time.Now()
	if err := cmd.commit(bs); err != nil {
		return err
	} else {
		elapsed := time.Since(s)
		cmd.log.Debug().Dur("elapsed", elapsed).Msg("committed")

		printCSV(cmd.Operations, "bench-commit", elapsed)
	}

	return cmd.checkNewBlock(blk)
}

func (cmd *BenchCommand) checkNewBlock(blk block.Block) error {
	switch ublk, found, err := cmd.local.Storage().Block(blk.Hash()); {
	case err != nil:
		return err
	case !found:
		return xerrors.Errorf("new block not found")
	default:
		if ublk.Operations().Empty() {
			return xerrors.Errorf("all operations not found; epty block.Operations()")
		}

		for _, op := range cmd.ops {
			if n, err := ublk.Operations().Get([]byte(op.Fact().Hash().String())); err != nil {
				return err
			} else if n == nil {
				err := xerrors.Errorf("operation not found")
				cmd.log.Error().Err(err).Hinted("operation", op.Fact().Hash()).Send()

				return err
			}
		}
	}

	return nil
}

func (cmd *BenchCommand) prepare() (ballot.Proposal, base.Voteproof, error) {
	max := uint(len(cmd.ops))
	_, _ = cmd.local.Policy().SetMaxOperationsInSeal(max)
	_, _ = cmd.local.Policy().SetMaxOperationsInProposal(max)

	if sl, err := operation.NewBaseSeal(
		cmd.senderPriv,
		cmd.ops,
		cmd.networkID,
	); err != nil {
		return nil, nil, err
	} else if err := cmd.storage.NewSeals([]seal.Seal{sl}); err != nil {
		return nil, nil, xerrors.Errorf("failed to store new seal: %w", err)
	}

	_ = cmd.dp.(logging.SetLogger).SetLogger(cmd.log)
	pm := isaac.NewProposalMaker(cmd.local)

	ib := cmd.newINITBallot(cmd.local)
	initFact := ib.INITBallotFactV0

	var proposal ballot.Proposal
	var ivp base.Voteproof
	if vp, err := cmd.newVoteproof(base.StageINIT, initFact); err != nil {
		return nil, nil, xerrors.Errorf("failed to make new voteproof: %w", err)
	} else {
		ivp = vp
	}

	if b, err := pm.Proposal(ivp.Round()); err != nil {
		return nil, nil, xerrors.Errorf("failed to make new proposal: %w", err)
	} else {
		proposal = b
	}

	if err := cmd.local.Storage().NewProposal(proposal); err != nil {
		return nil, nil, xerrors.Errorf("failed to store new proposal: %w", err)
	}

	return proposal, ivp, nil
}

func (cmd *BenchCommand) process(proposal ballot.Proposal, ivp base.Voteproof) (
	block.Block, storage.BlockStorage, error,
) {
	var blk block.Block
	if b, err := cmd.dp.ProcessINIT(proposal.Hash(), ivp); err != nil {
		return nil, nil, err
	} else {
		blk = b
	}

	acceptFact := ballot.NewACCEPTBallotV0(
		nil,
		ivp.Height(),
		ivp.Round(),
		proposal.Hash(),
		blk.Hash(),
		nil,
	).Fact()

	if avp, err := cmd.newVoteproof(base.StageACCEPT, acceptFact); err != nil {
		return nil, nil, xerrors.Errorf("failed to make new voteproof: %w", err)
	} else if bs, err := cmd.dp.ProcessACCEPT(proposal.Hash(), avp); err != nil {
		return nil, nil, xerrors.Errorf("failed to process accept voteproof: %w", err)
	} else {
		return blk, bs, nil
	}
}

func (cmd *BenchCommand) commit(bs storage.BlockStorage) error {
	if err := bs.Commit(context.Background()); err != nil {
		return xerrors.Errorf("failed to commit: %w", err)
	}

	return nil
}

func (cmd *BenchCommand) newOperation(
	sender base.Address,
	amount currency.Amount,
	keys currency.Keys,
	pks []key.Privatekey,
) (currency.CreateAccount, error) {
	token := util.UUID().Bytes()
	fact := currency.NewCreateAccountFact(token, sender, keys, amount)

	var fs []operation.FactSign
	for _, pk := range pks {
		if sig, err := operation.NewFactSignature(pk, fact, nil); err != nil {
			return currency.CreateAccount{}, err
		} else {
			fs = append(fs, operation.NewBaseFactSign(pk.Publickey(), sig))
		}
	}

	if ca, err := currency.NewCreateAccount(fact, fs, ""); err != nil {
		return currency.CreateAccount{}, err
	} else {
		return ca, nil
	}
}

func (cmd *BenchCommand) newOperations(n uint) ([]operation.Operation, error) {
	ops := make([]operation.Operation, n)
	for i := uint(0); i < n; i++ {
		keys, _ := currency.NewKeys([]currency.Key{currency.NewKey(key.MustNewBTCPrivatekey().Publickey(), 100)}, 100)
		if ca, err := cmd.newOperation(
			cmd.sender,
			currency.NewAmount(1),
			keys,
			[]key.Privatekey{cmd.senderPriv},
		); err != nil {
			return nil, err
		} else {
			ops[i] = ca
		}
	}

	return ops, nil
}

func (cmd *BenchCommand) newINITBallot(local *isaac.Localstate) ballot.INITBallotV0 {
	var ib ballot.INITBallotV0
	if b, err := isaac.NewINITBallotV0Round0(local.Storage(), local.Node().Address()); err != nil {
		panic(err)
	} else {
		ib = b
	}

	_ = ib.Sign(local.Node().Privatekey(), local.Policy().NetworkID())

	return ib
}

func (cmd *BenchCommand) newVoteproof(stage base.Stage, fact base.Fact) (base.VoteproofV0, error) {
	factHash := fact.Hash()

	var votes []base.VoteproofNodeFact
	if factSignature, err := cmd.local.Node().Privatekey().Sign(
		util.ConcatBytesSlice(
			factHash.Bytes(),
			cmd.local.Policy().NetworkID(),
		),
	); err != nil {
		return base.VoteproofV0{}, err
	} else {
		votes = []base.VoteproofNodeFact{base.NewVoteproofNodeFact(
			cmd.local.Node().Address(),
			valuehash.RandomSHA256(),
			factHash,
			factSignature,
			cmd.local.Node().Publickey(),
		)}
	}

	var height base.Height
	var round base.Round
	switch f := fact.(type) {
	case ballot.ACCEPTBallotFactV0:
		height = f.Height()
		round = f.Round()
	case ballot.INITBallotFactV0:
		height = f.Height()
		round = f.Round()
	}

	vp := base.NewVoteproofV0(
		height,
		round,
		cmd.suffrage.Nodes(),
		cmd.local.Policy().ThresholdRatio(),
		stage,
	)
	vp.SetResult(base.VoteResultMajority).
		SetMajority(fact).
		SetFacts([]base.Fact{fact}).
		SetVotes(votes)

	vp = *((&vp).Finish())

	return vp, nil
}

func (cmd *BenchCommand) localstate() (*isaac.Localstate, error) {
	var address currency.Address
	if addr, err := currency.NewAddress("bench"); err != nil {
		return nil, err
	} else {
		address = addr
	}
	cmd.log.Debug().Str("address", address.String()).Msg("address created")

	priv := key.MustNewBTCPrivatekey()
	cmd.log.Debug().Str("privatekey", priv.String()).Msg("private key of local node created")

	n := isaac.NewLocalNode(address, priv)
	cmd.log.Debug().Msg("local node created")

	local, err := isaac.NewLocalstate(cmd.storage, n, cmd.networkID)
	if err != nil {
		return nil, err
	} else if err := local.Initialize(); err != nil {
		return nil, err
	}
	cmd.log.Debug().Msg("localstate created")

	ks := []currency.Key{currency.NewKey(cmd.senderPriv.Publickey(), 100)}
	keys, _ := currency.NewKeys(ks, 100)
	cmd.sender, _ = currency.NewAddressFromKeys(ks)

	amount, _ := currency.NewAmountFromString(genesisAmountString)
	cmd.log.Debug().
		Str("amount", amount.String()).
		Str("privatekey", cmd.senderPriv.String()).
		Str("address", cmd.sender.String()).
		Msg("trying to create genesis account")
	if ga, err := currency.NewGenesisAccount(cmd.senderPriv, keys, amount, cmd.networkID); err != nil {
		return nil, err
	} else if genesis, err := isaac.NewGenesisBlockV0Generator(local, []operation.Operation{ga}); err != nil {
		return nil, err
	} else if _, err := genesis.Generate(); err != nil {
		return nil, err
	}
	cmd.log.Debug().Msg("genesis account generated")

	return local, nil
}

func (cmd *BenchCommand) prepareStorage() (storage.Storage, error) {
	uri := cmd.StorageURI
	uri.Path = fmt.Sprintf("bench_%s", util.UUID().String())

	client, err := mongodbstorage.NewClient(uri.String(), time.Second*2, time.Second*2)
	if err != nil {
		return nil, err
	}

	var benc encoder.Encoder
	if e, err := encs.Encoder(bsonenc.BSONType, ""); err != nil {
		return nil, err
	} else {
		benc = e
	}

	if st, err := mongodbstorage.NewStorage(client, encs, benc); err != nil {
		return nil, err
	} else if err := st.Initialize(); err != nil {
		return nil, err
	} else {
		return st, nil
	}
}

func printCSV(ops uint, s string, v interface{}) {
	var p interface{}
	switch t := v.(type) {
	case time.Duration:
		p = float64(t.Nanoseconds()) / 1000000000
	default:
		p = t
	}

	_, _ = fmt.Fprintf(os.Stdout, "%s,%d,%s,%f\n", localtime.RFC3339(time.Now()), ops, s, p)
}
