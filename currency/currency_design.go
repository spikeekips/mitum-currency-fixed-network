package currency

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/hint"
	"golang.org/x/xerrors"
)

var (
	CurrencyDesignType = hint.Type("mitum-currency-currency-design")
	CurrencyDesignHint = hint.NewHint(CurrencyDesignType, "v0.0.1")
)

type CurrencyDesign struct {
	Amount
	genesisAccount base.Address
	policy         CurrencyPolicy
}

func NewCurrencyDesign(amount Amount, genesisAccount base.Address, po CurrencyPolicy) CurrencyDesign {
	return CurrencyDesign{Amount: amount, genesisAccount: genesisAccount, policy: po}
}

func (de CurrencyDesign) IsValid([]byte) error {
	if err := de.Amount.IsValid(nil); err != nil {
		return xerrors.Errorf("invalid currency balance: %w", err)
	} else if !de.Big().OverZero() {
		return xerrors.Errorf("currency balance should be over zero")
	}

	if de.genesisAccount != nil {
		if err := de.genesisAccount.IsValid(nil); err != nil {
			return xerrors.Errorf("invalid CurrencyDesign: %w", err)
		}
	}

	if err := de.policy.IsValid(nil); err != nil {
		return xerrors.Errorf("invalid CurrencyPolicy: %w", err)
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
