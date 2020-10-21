// +build mongodb

package digest

import (
	"context"

	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (t *testStorage) TestBlockStorageWithOperations() {
	addrs := make([]base.Address, 5)
	for i := 0; i < len(addrs); i++ {
		addrs[i] = currency.MustAddress(util.UUID().String())
	}

	// 10 operations with different address
	ops := make([]operation.Operation, len(addrs))
	opsByAddress := map[string][]string{}
	for i := 0; i < len(addrs); i++ {
		sender := addrs[i]

		var receiver base.Address
		if len(addrs) == i+1 {
			receiver = addrs[0]
		} else {
			receiver = addrs[i+1]
		}

		op := t.newTransfer(sender, receiver)
		ops[i] = op

		opsByAddress[sender.String()] = append(opsByAddress[sender.String()], op.Fact().Hash().String())
		opsByAddress[receiver.String()] = append(opsByAddress[receiver.String()], op.Fact().Hash().String())
	}

	blk, err := block.NewBlockV0(
		block.SuffrageInfoV0{},
		base.Height(3),
		base.Round(1),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		localtime.Now(),
	)
	t.NoError(err)

	nblk := blk.SetOperations(ops)

	st, _ := t.Storage()
	bs, err := st.BlockStorage(nblk)
	t.NoError(err)

	t.NoError(bs.Prepare())
	t.NoError(bs.Commit(context.Background()))

	for _, op := range ops {
		va, found, err := st.Operation(op.Fact().Hash(), true)
		t.NoError(err)
		t.True(found)

		t.True(op.Hash().Equal(va.Operation().Hash()))
		t.True(op.Fact().Hash().Equal(va.Operation().Fact().Hash()))
		t.True(localtime.Equal(blk.ConfirmedAt(), va.ConfirmedAt()))
	}

	for _, address := range addrs {
		var i int
		ops := make([]string, len(opsByAddress[address.String()]))
		err := st.OperationsByAddress(address, true, false, "", 0, func(fh valuehash.Hash, va OperationValue) (bool, error) {
			ops[i] = va.Operation().Fact().Hash().String()
			i++

			return true, nil
		})
		t.NoError(err)

		opsa := opsByAddress[address.String()]

		t.Equal(opsa, ops)

		// reverse
		ops = make([]string, len(opsByAddress[address.String()]))
		i = len(ops) - 1
		err = st.OperationsByAddress(address, true, true, "", 0, func(fh valuehash.Hash, va OperationValue) (bool, error) {
			ops[i] = va.Operation().Fact().Hash().String()
			i--

			return true, nil
		})
		t.NoError(err)

		t.Equal(opsa, ops)
	}
}

func (t *testStorage) TestBlockStorageWithStates() {
	blk, err := block.NewBlockV0(
		block.SuffrageInfoV0{},
		base.Height(3),
		base.Round(1),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		localtime.Now(),
	)
	t.NoError(err)

	// 10 accounts
	acs := make([]currency.Account, 5)
	sts := make([]state.State, len(acs)*2)
	balances := map[string]currency.Amount{}
	for i := 0; i < len(acs); i++ {
		ac := t.newAccount()
		acs[i] = ac
		sta := t.newAccountState(ac, blk.Height())

		am := t.randomAmount()
		stb := t.newBalanceState(ac, blk.Height(), am)

		balances[ac.Address().String()] = am

		sts[i*2] = sta
		sts[i*2+1] = stb
	}

	nblk := blk.SetStates(sts)
	st, _ := t.Storage()
	bs, err := st.BlockStorage(nblk)
	t.NoError(err)

	t.NoError(bs.Prepare())
	t.NoError(bs.Commit(context.Background()))

	for _, ac := range acs {
		uac, found, err := st.Account(ac.Address())
		t.NoError(err)
		t.True(found)

		t.True(ac.Address().Equal(uac.Account().Address()))
		t.Equal(blk.Height(), uac.Height())
		t.Equal(blk.Height()-1, uac.PreviousHeight())
		t.Equal(balances[ac.Address().String()], uac.Balance())
	}
}
