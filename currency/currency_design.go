package currency

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	CurrencyDesignType   = hint.Type("mitum-currency-currency-design")
	CurrencyDesignHint   = hint.NewHint(CurrencyDesignType, "v0.0.1")
	CurrencyDesignHinter = CurrencyDesign{BaseHinter: hint.NewBaseHinter(CurrencyDesignHint)}
)

type CurrencyDesign struct {
	hint.BaseHinter
	Amount
	genesisAccount base.Address
	policy         CurrencyPolicy
	aggregate      Big
}

func NewCurrencyDesign(amount Amount, genesisAccount base.Address, po CurrencyPolicy) CurrencyDesign {
	return CurrencyDesign{
		BaseHinter:     hint.NewBaseHinter(CurrencyDesignHint),
		Amount:         amount,
		genesisAccount: genesisAccount,
		policy:         po,
		aggregate:      amount.Big(),
	}
}

func (de CurrencyDesign) IsValid([]byte) error {
	if err := isvalid.Check(nil, false,
		de.BaseHinter,
		de.Amount,
		de.aggregate,
	); err != nil {
		return isvalid.InvalidError.Errorf("invalid currency balance: %w", err)
	}

	switch {
	case !de.Big().OverZero():
		return isvalid.InvalidError.Errorf("currency balance should be over zero")
	case !de.aggregate.OverZero():
		return isvalid.InvalidError.Errorf("aggregate should be over zero")
	}

	if de.genesisAccount != nil {
		if err := de.genesisAccount.IsValid(nil); err != nil {
			return isvalid.InvalidError.Errorf("invalid CurrencyDesign: %w", err)
		}
	}

	if err := de.policy.IsValid(nil); err != nil {
		return isvalid.InvalidError.Errorf("invalid CurrencyPolicy: %w", err)
	}

	return nil
}

func (de CurrencyDesign) Bytes() []byte {
	var gb []byte
	if de.genesisAccount != nil {
		gb = de.genesisAccount.Bytes()
	}

	return util.ConcatBytesSlice(
		de.Amount.Bytes(),
		gb,
		de.policy.Bytes(),
		de.aggregate.Bytes(),
	)
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
