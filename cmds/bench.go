package cmds

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	contestlib "github.com/spikeekips/mitum/contest/lib"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/localfs"
	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

var genesisAmountString = "99999999999999999999999999"

type BenchCommand struct {
	BaseCommand
	StorageURI    *url.URL `name:"storage" help:"mongodb storage uri (default:mongodb://localhost:27017)" default:"mongodb://localhost:27017"` // nolint
	Operations    uint     `arg:"" name:"operations" help:"number of operations (default:10)" default:"10"`
	priv          key.Privatekey
	networkID     base.NetworkID
	storage       storage.Storage
	genesis       *account
	lastHeight    base.Height
	local         *isaac.Local
	suffrage      base.Suffrage
	accounts      []*account
	ops           []operation.Operation
	opsExclude    []operation.Operation
	ivp           base.Voteproof
	proposal      ballot.Proposal
	block         block.Block
	previousBlock block.Block
	fa            currency.FeeAmount
}

func (cmd *BenchCommand) Run(flags *MainFlags, version util.Version, l logging.Logger) error {
	_ = cmd.BaseCommand.Run(flags, version, l)

	if l, err := SetupLogging(flags.Log, flags.LogFlags); err != nil {
		return err
	} else {
		cmd.log = l
	}

	if cmd.Operations < 4 {
		return xerrors.Errorf("operations should be over 4")
	}

	cmd.Log().Info().Str("version", version.String()).Msg("mitum-currency")
	cmd.Log().Debug().Interface("flags", flags).Msg("flags parsed")
	defer cmd.Log().Info().Msg("mitum-currency finished")

	cmd.Log().Info().Msg("trying to benchmark")

	cmd.genesis = newAccount()

	return cmd.run()
}

func (cmd *BenchCommand) run() error {
	cmd.priv = key.MustNewBTCPrivatekey()
	cmd.networkID = util.UUID().Bytes()

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
	cmd.Log().Debug().Msg("storage prepared")

	if local, err := cmd.makeLocal(); err != nil {
		return xerrors.Errorf("failed to preaper local: %w", err)
	} else {
		cmd.local = local
	}
	cmd.Log().Debug().Msg("local prepared")

	cmd.suffrage = base.NewFixedSuffrage(cmd.local.Node().Address(), []base.Address{cmd.local.Node().Address()})
	if err := cmd.suffrage.Initialize(); err != nil {
		return xerrors.Errorf("failed to initialize suffrage: %w", err)
	}
	cmd.Log().Debug().Msg("fixed suffrage prepared")

	if err := cmd.elapsed("prepare-accounts", cmd.prepareAccounts); err != nil {
		return xerrors.Errorf("failed to prepare accounts: %w", err)
	}
	cmd.Log().Debug().Int("accounts", len(cmd.accounts)).Msg("accounts prepared")

	if err := cmd.elapsed("prepare-operations", cmd.prepareOperations); err != nil {
		return xerrors.Errorf("failed to prepare operations: %w", err)
	}
	cmd.Log().Debug().
		Int("operations", len(cmd.ops)).Int("excluded_operations", len(cmd.opsExclude)).
		Msg("operations prepared")

	if err := cmd.elapsed("prepare-processor", cmd.prepareProcessor); err != nil {
		return xerrors.Errorf("failed to prepare processor: %w", err)
	}
	cmd.Log().Debug().Msg("processor prepared")

	cmd.Log().Debug().Msg("running again for checking block hash")
	for i := 0; i < 10; i++ {
		if cmd.previousBlock != nil {
			if err := cmd.clean(); err != nil {
				panic(err)
			}
		}

		if err := cmd.try(i); err != nil {
			return err
		}
	}

	return nil
}

func (cmd *BenchCommand) clean() error {
	if err := cmd.local.Storage().CleanByHeight(cmd.previousBlock.Height()); err != nil {
		return err
	}

	if err := cmd.local.BlockFS().CleanByHeight(cmd.previousBlock.Height()); err != nil {
		return err
	}

	return nil
}

func (cmd *BenchCommand) try(i int) error {
	if err := cmd.elapsed("process", cmd.process); err != nil {
		return xerrors.Errorf("failed to process: %w", err)
	}

	if err := cmd.elapsed("check-result", cmd.checkProcess); err != nil {
		return xerrors.Errorf("failed to check result: %w", err)
	}

	if !cmd.previousBlock.Hash().Equal(cmd.block.Hash()) {
		cmd.Log().Error().Int("try", i).
			Dict("previous_block", logging.Dict().
				Hinted("hash", cmd.previousBlock.Hash()).
				Hinted("states", cmd.previousBlock.StatesHash()).
				Hinted("operations", cmd.previousBlock.OperationsHash()),
			).
			Dict("block", logging.Dict().
				Hinted("hash", cmd.block.Hash()).
				Hinted("states", cmd.block.StatesHash()).
				Hinted("operations", cmd.block.OperationsHash()),
			).
			Msg("block hash does not matched")

		return xerrors.Errorf("block hash does not match; %v != %v", cmd.previousBlock.Hash(), cmd.block.Hash())
	} else {
		cmd.Log().Info().Int("try", i).
			Hinted("previous_block", cmd.previousBlock.Hash()).
			Hinted("new_block", cmd.block.Hash()).
			Msg("block hash matched")
	}

	cmd.Log().Debug().Msg("checked")

	return nil
}

func (cmd *BenchCommand) prepareStorage() (storage.Storage, error) {
	uri := cmd.StorageURI
	uri.Path = fmt.Sprintf("bench_%s", util.UUID().String())

	client, err := mongodbstorage.NewClient(uri.String(), time.Second*2, time.Second*2)
	if err != nil {
		return nil, err
	}

	var ca cache.Cache
	if c, err := cache.NewGCache("lru", 100*100, time.Minute*3); err != nil {
		return nil, err
	} else {
		ca = c
	}

	if st, err := mongodbstorage.NewStorage(client, encs, nil, ca); err != nil {
		return nil, err
	} else if err := st.Initialize(); err != nil {
		return nil, err
	} else {
		return st, nil
	}
}

func (cmd *BenchCommand) makeLocal() (*isaac.Local, error) {
	var address currency.Address
	if addr, err := currency.NewAddress("bench"); err != nil {
		return nil, err
	} else {
		address = addr
	}

	n := isaac.NewLocalNode(address, key.MustNewBTCPrivatekey(), "quic://local")
	cmd.Log().Debug().Msg("local node created")

	local, err := isaac.NewLocal(cmd.storage, localfs.TempBlockFS(defaultJSONEnc), n, cmd.networkID)
	if err != nil {
		return nil, err
	}

	contestlib.ExitHooks.Add(func() {
		if err := local.BlockFS().Clean(true); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
		}
	})

	if err := local.Initialize(); err != nil {
		return nil, err
	}
	cmd.Log().Debug().Msg("local created")

	amount, _ := currency.NewAmountFromString(genesisAmountString)
	cmd.Log().Debug().
		Str("amount", amount.String()).
		Str("privatekey", cmd.genesis.Priv.String()).
		Str("address", cmd.genesis.Address.String()).
		Msg("trying to create genesis account")
	if ga, err := currency.NewGenesisAccount(cmd.genesis.Priv, cmd.genesis.Keys, amount, cmd.networkID); err != nil {
		return nil, err
	} else if genesis, err := isaac.NewGenesisBlockV0Generator(local, []operation.Operation{ga}); err != nil {
		return nil, err
	} else if _, err := genesis.Generate(); err != nil {
		return nil, err
	}
	cmd.Log().Debug().Msg("genesis account generated")

	_, _ = local.Policy().SetMaxOperationsInProposal(cmd.Operations + 10)

	switch m, found, err := cmd.storage.LastManifest(); {
	case err != nil:
		return nil, err
	case !found:
		return nil, xerrors.Errorf("last block not found")
	default:
		cmd.lastHeight = m.Height()
	}

	return local, nil
}

type account struct {
	Priv    key.Privatekey
	Address base.Address
	Keys    currency.Keys
}

func newAccount() *account {
	priv := key.MustNewBTCPrivatekey()

	if k, err := currency.NewKey(priv.Publickey(), 100); err != nil {
		panic(err)
	} else if keys, err := currency.NewKeys([]currency.Key{k}, 100); err != nil {
		panic(err)
	} else {
		address, _ := currency.NewAddressFromKeys(keys)

		return &account{
			Priv:    priv,
			Address: address,
			Keys:    keys,
		}
	}
}

type acerr struct {
	err error
	ac  interface{}
	sts []state.State
}

func (ac acerr) Error() string {
	return ac.err.Error()
}

func (cmd *BenchCommand) prepareAccounts() error {
	var n uint = 100
	if n > cmd.Operations {
		n = cmd.Operations
	}

	cmd.Log().Debug().Uint("number_of_accounts", cmd.Operations).Uint("workers", n).Msg("preparing to create accounts")

	errchan := make(chan error)
	wk := util.NewDistributeWorker(n, errchan)

	go func() {
		_ = wk.Run(
			func(i uint, j interface{}) error {
				if j == nil {
					return nil
				}
				ac, sts, err := cmd.createAccount(currency.NewAmount(int64(100)))

				return acerr{err: err, ac: ac, sts: sts}
			},
		)

		close(errchan)
	}()

	go func() {
		for i := 0; i < int(cmd.Operations)+1; i++ {
			wk.NewJob(i)
		}
		wk.Done(true)
	}()

	acs := make([]*account, int(cmd.Operations)+1)
	var i int
	for err := range errchan {
		if err == nil {
			continue
		}

		var aerr acerr
		if !xerrors.As(err, &aerr) {
			return err
		} else if aerr.err != nil {
			return err
		}

		acs[i] = aerr.ac.(*account)

		for _, st := range aerr.sts {
			if err := cmd.storage.(storage.StateUpdater).NewState(st); err != nil {
				return err
			}
		}
		i++
	}

	cmd.accounts = acs

	return nil
}

func (cmd *BenchCommand) createAccount(amount currency.Amount) (*account, []state.State, error) {
	ac := newAccount()

	sts := make([]state.State, 2)

	{
		key := currency.StateKeyAccount(ac.Address)
		var stu *state.StateUpdater
		if st, err := state.NewStateV0(key, nil, base.NilHeight); err != nil {
			return nil, nil, err
		} else {
			if ac, err := currency.NewAccountFromKeys(ac.Keys); err != nil {
				return nil, nil, err
			} else if nst, err := currency.SetStateAccountValue(st, ac); err != nil {
				return nil, nil, err
			} else {
				stu = state.NewStateUpdater(nst)
			}
		}

		fh := valuehash.RandomSHA256()
		cmd.log.Debug().Hinted("op", fh).Msg("op account")
		if err := stu.AddOperation(fh); err != nil {
			return nil, nil, err
		} else {
			stu = stu.SetHeight(cmd.lastHeight)
			if err := stu.SetHash(stu.GenerateHash()); err != nil {
				return nil, nil, err
			}

			sts[0] = stu.GetState()
		}
	}

	{
		key := currency.StateKeyBalance(ac.Address)
		value, _ := state.NewStringValue(amount.String())

		var stu *state.StateUpdater
		if st, err := state.NewStateV0(key, value, base.NilHeight); err != nil {
			return nil, nil, err
		} else {
			stu = state.NewStateUpdater(st)
		}

		fh := valuehash.RandomSHA256()
		cmd.log.Debug().Hinted("op", fh).Msg("op balance")
		if err := stu.SetHash(stu.GenerateHash()); err != nil {
			return nil, nil, err
		} else if err := stu.AddOperation(fh); err != nil {
			return nil, nil, err
		} else {
			stu = stu.SetHeight(cmd.lastHeight)
			if err := stu.SetHash(stu.GenerateHash()); err != nil {
				return nil, nil, err
			}
			sts[1] = stu.GetState()
		}
	}

	return ac, sts, nil
}

func (cmd *BenchCommand) prepareOperations() error { // nolint:funlen
	var n uint = 100
	if n > cmd.Operations {
		n = cmd.Operations
	}

	cmd.Log().Debug().Uint("number_of_operations", cmd.Operations).Uint("workers", n).Msg("preparing to create accounts")

	errchan := make(chan error, cmd.Operations)
	wk := util.NewDistributeWorker(n, errchan)

	go func() {
		_ = wk.Run(
			func(_ uint, j interface{}) error {
				if j == nil {
					return nil
				}

				i := j.(int)

				sender := cmd.accounts[i]
				var receiver *account
				if len(cmd.accounts) == i+1 {
					receiver = cmd.accounts[0]
				} else {
					receiver = cmd.accounts[i+1]
				}
				op, err := cmd.newOperation(sender, receiver, currency.NewAmount(1))

				return acerr{err: err, ac: op}
			},
		)
		close(errchan)
	}()

	go func() {
		for i := 0; i < int(cmd.Operations); i++ {
			wk.NewJob(i)
		}
		wk.Done(true)
	}()

	ops := make([]operation.Operation, cmd.Operations)
	var i int
	for err := range errchan {
		if err == nil {
			continue
		}

		var aerr acerr
		if !xerrors.As(err, &aerr) {
			return err
		} else if aerr.err != nil {
			return err
		}

		ops[i] = aerr.ac.(operation.Operation)
		cmd.log.Debug().Hinted("op", ops[i].Fact().Hash()).Msg("op")
		i++
	}

	if err := cmd.prepareExcludeOperations(); err != nil {
		return err
	}

	cmd.ops = ops

	return cmd.elapsed("generate-seals", cmd.generateSeals)
}

func (cmd *BenchCommand) prepareExcludeOperations() error {
	excludes := make([]operation.Operation, 2)
	for i := range cmd.accounts[:2] {
		op, err := cmd.newOperation(cmd.accounts[i], cmd.accounts[i+1], currency.NewAmount(1))
		if err != nil {
			return err
		}

		cmd.log.Debug().Hinted("op", op.Fact().Hash()).Msg("exclude")
		excludes[i] = op
	}

	cmd.opsExclude = excludes

	return nil
}

func (cmd *BenchCommand) generateSeals() error {
	l := int(cmd.local.Policy().MaxOperationsInSeal())
	var ops []operation.Operation // nolint: prealloc
	for i := range cmd.ops {
		ops = append(ops, cmd.ops[i])
		if len(ops) == l {
			if err := cmd.generateSeal(ops); err != nil {
				return err
			}

			ops = nil
		}
	}

	if len(ops) > 0 {
		if err := cmd.generateSeal(ops); err != nil {
			return err
		}
	}

	return cmd.generateSeal(cmd.opsExclude)
}

func (cmd *BenchCommand) generateSeal(ops []operation.Operation) error {
	if sl, err := operation.NewBaseSeal(cmd.priv, ops, cmd.networkID); err != nil {
		return err
	} else if err := cmd.storage.NewSeals([]seal.Seal{sl}); err != nil {
		return xerrors.Errorf("failed to store new seal: %w", err)
	} else {
		return nil
	}
}

func (cmd *BenchCommand) newOperation(sender, receiver *account, amount currency.Amount) (currency.Transfers, error) {
	token := util.UUID().Bytes()
	item := currency.NewTransferItem(receiver.Address, amount)
	fact := currency.NewTransfersFact(
		token,
		sender.Address,
		[]currency.TransferItem{item},
	)

	var fs []operation.FactSign
	if sig, err := operation.NewFactSignature(sender.Priv, fact, nil); err != nil {
		return currency.Transfers{}, err
	} else {
		fs = append(fs, operation.NewBaseFactSign(sender.Priv.Publickey(), sig))
	}

	if tf, err := currency.NewTransfers(fact, fs, ""); err != nil {
		return currency.Transfers{}, err
	} else {
		return tf, nil
	}
}

func (cmd *BenchCommand) prepareProcessor() error {
	pm := isaac.NewProposalMaker(cmd.local)

	ib := cmd.newINITBallot()
	initFact := ib.INITBallotFactV0

	if vp, err := cmd.newVoteproof(base.StageINIT, initFact); err != nil {
		return xerrors.Errorf("failed to make new voteproof: %w", err)
	} else {
		cmd.ivp = vp
	}

	if b, err := pm.Proposal(cmd.ivp.Round()); err != nil {
		return xerrors.Errorf("failed to make new proposal: %w", err)
	} else if err := cmd.local.Storage().NewProposal(b); err != nil {
		return xerrors.Errorf("failed to store new proposal: %w", err)
	} else {
		cmd.proposal = b
	}

	cmd.Log().Debug().Int("seals", len(cmd.proposal.Seals())).Msg("proposal created")

	cmd.fa = currency.NewFixedFeeAmount(currency.NewAmount(1))

	return nil
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

func (cmd *BenchCommand) newINITBallot() ballot.INITBallotV0 {
	var ib ballot.INITBallotV0
	if b, err := isaac.NewINITBallotV0Round0(cmd.local); err != nil {
		panic(err)
	} else {
		ib = b
	}

	_ = ib.Sign(cmd.local.Node().Privatekey(), cmd.local.Policy().NetworkID())

	return ib
}

func (cmd *BenchCommand) dp() (*isaac.DefaultProposalProcessor, error) {
	dp := isaac.NewDefaultProposalProcessor(cmd.local, cmd.suffrage)
	_ = dp.SetLogger(cmd.Log())

	opr := currency.NewOperationProcessor(cmd.fa, func() (base.Address, error) {
		return cmd.accounts[0].Address, nil
	})

	if _, err := dp.AddOperationProcessor(currency.Transfers{}, opr); err != nil {
		return nil, err
	} else if _, err := dp.AddOperationProcessor(currency.CreateAccounts{}, opr); err != nil {
		return nil, err
	}

	return dp, nil
}

func (cmd *BenchCommand) process() error {
	var dp *isaac.DefaultProposalProcessor
	if d, err := cmd.dp(); err != nil {
		return err
	} else {
		dp = d
	}

	started := time.Now()
	var blk block.Block
	if b, err := dp.ProcessINIT(cmd.proposal.Hash(), cmd.ivp); err != nil {
		return err
	} else {
		cmd.printElapsed("process-init", started)

		blk = b
	}

	cmd.Log().Debug().Msg("init processed")

	acceptFact := ballot.NewACCEPTBallotV0(
		nil,
		cmd.ivp.Height(), cmd.ivp.Round(),
		cmd.proposal.Hash(), blk.Hash(),
		nil,
	).Fact()

	started = time.Now()
	var bs storage.BlockStorage
	if avp, err := cmd.newVoteproof(base.StageACCEPT, acceptFact); err != nil {
		return xerrors.Errorf("failed to make new voteproof: %w", err)
	} else if s, err := dp.ProcessACCEPT(cmd.proposal.Hash(), avp); err != nil {
		return xerrors.Errorf("failed to process accept voteproof: %w", err)
	} else {
		cmd.printElapsed("process-accept", started)

		bs = s
	}
	cmd.Log().Debug().Msg("acccept processed")

	for k, v := range dp.States() {
		printCSV(cmd.Operations, "pp-"+k, v)
	}

	cmd.Log().Debug().Msg("trying to commit")
	if err := cmd.commit(bs); err != nil {
		return xerrors.Errorf("failed to commit: %w", err)
	}
	cmd.Log().Debug().Msg("committed")

	if cmd.block == nil {
		cmd.previousBlock = blk
	}

	cmd.block = blk

	cmd.Log().Debug().Hinted("height", blk.Height()).Hinted("block", blk.Hash()).Msg("new block")

	return nil
}

func (cmd *BenchCommand) commit(bs storage.BlockStorage) error {
	started := time.Now()
	defer func() {
		cmd.printElapsed("commit", started)

		for k, v := range bs.States() {
			printCSV(cmd.Operations, "bs-"+k, v)
		}
	}()

	return bs.Commit(context.Background())
}

func (cmd *BenchCommand) checkProcess() error { // nolint:funlen
	facts := map[string]valuehash.Hash{}
	for i := range cmd.ops {
		fh := cmd.ops[i].Fact().Hash()
		facts[fh.String()] = fh
	}

	excludes := map[string]valuehash.Hash{}
	for i := range cmd.opsExclude {
		fh := cmd.opsExclude[i].Fact().Hash()
		facts[fh.String()] = fh
		excludes[fh.String()] = fh
	}

	feeOps := map[string]struct{}{}
	for i := range cmd.block.Operations() {
		op := cmd.block.Operations()[i]
		if _, ok := op.(currency.FeeOperation); !ok {
			continue
		}

		feeOps[op.Fact().Hash().String()] = struct{}{}
	}

	var founds int
	var notFounds, notInStates, inStates []valuehash.Hash
	_ = cmd.block.OperationsTree().Traverse(func(i int, key, _, v []byte) (bool, error) {
		fh := valuehash.NewBytes(key)

		if _, found := facts[fh.String()]; !found {
			if _, found := feeOps[fh.String()]; !found {
				notFounds = append(notFounds, fh)
				cmd.Log().Error().Hinted("fact", fh).Msg("fact not found in operation tree")
			}
		} else {
			founds++
		}
		_, inExcludes := excludes[fh.String()]

		switch mod, err := base.BytesToFactMode(v); {
		case err != nil:
			cmd.Log().Error().Err(err).Hinted("fact", fh).Bytes("mod", v).Msg("invalid FactMode found")
		case mod&base.FInStates == 0:
			if !inExcludes {
				notInStates = append(notInStates, fh)
				cmd.Log().Error().Hinted("fact", fh).Bytes("mod", v).Msg("fact not found in states tree")
			}
		case inExcludes:
			inStates = append(inStates, fh)
			cmd.Log().Error().Hinted("fact", fh).Bytes("mod", v).Msg("fact should not found in states tree")
		}
		return true, nil
	})

	if n := len(notFounds); n > 0 {
		cmd.Log().Error().Int("not_founds", n).Msg("not found in OperationsTree")
	}
	if n := len(notInStates); n > 0 {
		cmd.Log().Error().Int("not_in_states", n).Msg("not found in states")
	}
	if n := len(inStates); n > 0 {
		cmd.Log().Error().Int("in_states", n).Msg("found in states")
	}

	if founds != len(facts) {
		if len(notFounds) > 0 || len(notInStates) > 0 || len(inStates) > 0 {
			return xerrors.Errorf("failed to process")
		}
	}

	cmd.Log().Info().Msg("all operations in states")

	return nil
}

func (cmd *BenchCommand) elapsed(s string, f func() error) error {
	started := time.Now()
	defer func() {
		cmd.printElapsed(s, started)
	}()

	return f()
}

func (cmd *BenchCommand) printElapsed(s string, started time.Time) {
	printCSV(cmd.Operations, s, time.Since(started))
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
