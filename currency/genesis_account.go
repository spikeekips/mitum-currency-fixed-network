package currency

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	GenesisAccountFactType = hint.MustNewType(0xa0, 0x07, "mitum-currency-genesis-account-operation-fact")
	GenesisAccountFactHint = hint.MustHint(GenesisAccountFactType, "0.0.1")
	GenesisAccountType     = hint.MustNewType(0xa0, 0x08, "mitum-currency-genesis-account-operation")
	GenesisAccountHint     = hint.MustHint(GenesisAccountType, "0.0.1")
)

type GenesisAccountFact struct {
	h              valuehash.Hash
	token          []byte
	genesisNodeKey key.Publickey
	keys           Keys
	amount         Amount
}

func NewGenesisAccountFact(token []byte, genesisNodeKey key.Publickey, keys Keys, amount Amount) GenesisAccountFact {
	gaf := GenesisAccountFact{
		token:          token,
		genesisNodeKey: genesisNodeKey,
		keys:           keys,
		amount:         amount,
	}
	gaf.h = valuehash.NewSHA256(gaf.Bytes())

	return gaf
}

func (gaf GenesisAccountFact) Hint() hint.Hint {
	return GenesisAccountFactHint
}

func (gaf GenesisAccountFact) Hash() valuehash.Hash {
	return gaf.h
}

func (gaf GenesisAccountFact) Bytes() []byte {
	return util.ConcatBytesSlice(
		gaf.token,
		[]byte(gaf.genesisNodeKey.String()),
		gaf.keys.Bytes(),
		gaf.amount.Bytes(),
	)
}

func (gaf GenesisAccountFact) IsValid([]byte) error {
	if len(gaf.token) < 1 {
		return xerrors.Errorf("empty token for GenesisAccountFact")
	}

	return isvalid.Check([]isvalid.IsValider{
		gaf.h,
		gaf.genesisNodeKey,
		gaf.keys,
		gaf.amount,
	}, nil, false)
}

func (gaf GenesisAccountFact) Token() []byte {
	return gaf.token
}

func (gaf GenesisAccountFact) GenesisNodeKey() key.Publickey {
	return gaf.genesisNodeKey
}

func (gaf GenesisAccountFact) Amount() Amount {
	return gaf.amount
}

func (gaf GenesisAccountFact) Keys() Keys {
	return gaf.keys
}

func (gaf GenesisAccountFact) Address() (base.Address, error) {
	return NewAddressFromKeys(gaf.keys)
}

func (gaf GenesisAccountFact) Addresses() ([]base.Address, error) {
	if a, err := gaf.Address(); err != nil {
		return nil, err
	} else {
		return []base.Address{a}, nil
	}
}

type GenesisAccount struct {
	operation.BaseOperation
}

func NewGenesisAccount(
	genesisNodeKey key.Privatekey,
	keys Keys,
	amount Amount,
	networkID base.NetworkID,
) (GenesisAccount, error) {
	fact := NewGenesisAccountFact(networkID, genesisNodeKey.Publickey(), keys, amount)

	var fs []operation.FactSign
	if sig, err := operation.NewFactSignature(genesisNodeKey, fact, networkID); err != nil {
		return GenesisAccount{}, err
	} else {
		fs = []operation.FactSign{operation.NewBaseFactSign(genesisNodeKey.Publickey(), sig)}
	}

	if bo, err := operation.NewBaseOperationFromFact(GenesisAccountHint, fact, fs); err != nil {
		return GenesisAccount{}, err
	} else {
		return GenesisAccount{BaseOperation: bo}, nil
	}
}

func (ga GenesisAccount) Hint() hint.Hint {
	return GenesisAccountHint
}

func (ga GenesisAccount) IsValid(networkID []byte) error {
	if err := operation.IsValidOperation(ga, networkID); err != nil {
		return err
	}

	if len(ga.Signs()) != 1 {
		return xerrors.Errorf("genesis account should be signed only by genesis node key")
	}

	fact := ga.Fact().(GenesisAccountFact)
	if !fact.genesisNodeKey.Equal(ga.Signs()[0].Signer()) {
		return xerrors.Errorf("not signed by genesis node key")
	}

	return nil
}

func (ga GenesisAccount) Process(
	getState func(key string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) error {
	fact := ga.Fact().(GenesisAccountFact)

	var newAddress base.Address
	if a, err := fact.Address(); err != nil {
		return util.IgnoreError.Wrap(err)
	} else {
		newAddress = a
	}

	var ns, nb state.State
	if st, err := notExistsAccountState(StateKeyAccount(newAddress), "key of genesis", getState); err != nil {
		return err
	} else {
		ns = st
	}

	if st, err := notExistsAccountState(StateKeyBalance(newAddress), "balance of genesis", getState); err != nil {
		return err
	} else {
		nb = st
	}

	var nac Account
	if ac, err := NewAccountFromKeys(fact.keys); err != nil {
		return err
	} else {
		nac = ac
	}

	if st, err := SetStateAccountValue(ns, nac); err != nil {
		return util.IgnoreError.Wrap(err)
	} else {
		ns = st
	}

	if st, err := SetStateAmountValue(nb, fact.amount); err != nil {
		return util.IgnoreError.Wrap(err)
	} else {
		nb = st
	}

	return setState(fact.Hash(), ns, nb)
}
