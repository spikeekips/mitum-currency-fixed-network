package currency

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	CurrencyDesignType = hint.Type("mitum-currency-currency-design")
	CurrencyDesignHint = hint.NewHint(CurrencyDesignType, "v0.0.1")
)

type CurrencyDesign struct {
	Amount
	genesisAccount base.Address
	policy         CurrencyPolicy
	aggregate      Big
}

func NewCurrencyDesign(amount Amount, genesisAccount base.Address, po CurrencyPolicy) CurrencyDesign {
	return CurrencyDesign{Amount: amount, genesisAccount: genesisAccount, policy: po, aggregate: amount.Big()}
}

func (de CurrencyDesign) IsValid([]byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		de.Amount,
		de.aggregate,
	}, nil, false); err != nil {
		return errors.Wrap(err, "invalid currency balance")
	}

	switch {
	case !de.Big().OverZero():
		return errors.Errorf("currency balance should be over zero")
	case !de.aggregate.OverZero():
		return errors.Errorf("aggregate should be over zero")
	}

	if de.genesisAccount != nil {
		if err := de.genesisAccount.IsValid(nil); err != nil {
			return errors.Wrap(err, "invalid CurrencyDesign")
		}
	}

	if err := de.policy.IsValid(nil); err != nil {
		return errors.Wrap(err, "invalid CurrencyPolicy")
	}

	return nil
}

func (CurrencyDesign) Hint() hint.Hint {
	return CurrencyDesignHint
}

func (de CurrencyDesign) GenesisAccount() base.Address {
	return de.genesisAccount
}

func (de CurrencyDesign) Policy() CurrencyPolicy {
	return de.policy
}

func (de CurrencyDesign) SetPolicy(po CurrencyPolicy) CurrencyDesign {
	de.policy = po

	return de
}

func (de CurrencyDesign) Aggregate() Big {
	return de.aggregate
}

func (de CurrencyDesign) AddAggregate(b Big) (CurrencyDesign, error) {
	if !b.OverZero() {
		return de, errors.Errorf("new aggregate not over zero")
	}

	de.aggregate = de.aggregate.Add(b)

	return de, nil
}
