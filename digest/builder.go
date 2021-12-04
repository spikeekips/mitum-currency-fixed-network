package digest

import (
	"bytes"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum-currency/currency"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
)

var (
	templatePrivateKeyString = "Kzxb3TcaxHCp9iq6ekyaNjaeRSdqzvv9JrazTV8cVsZq9U2FQSSG"
	templatePublickey        key.Publickey
	templateCurrencyID       = currency.CurrencyID("xXx")
	templateSender           = currency.Address("mother")
	templateReceiver         = currency.Address("father")
	templateToken            = []byte("raised by")
	templateSignature        = key.Signature([]byte("wolves"))
	templateBig              = currency.NewBig(-333)
	templateSignedAtString   = "2020-10-08T07:53:26Z"
	templateSignedAt         time.Time
)

func init() {
	if priv, err := key.NewBTCPrivatekeyFromString(templatePrivateKeyString); err != nil {
		panic(err)
	} else {
		templatePublickey = priv.Publickey()
	}

	templateSignedAt, _ = time.Parse(time.RFC3339, templateSignedAtString)
}

type Builder struct {
	enc       encoder.Encoder
	networkID base.NetworkID
}

func NewBuilder(enc encoder.Encoder, networkID base.NetworkID) Builder {
	return Builder{enc: enc, networkID: networkID}
}

func (bl Builder) FactTemplate(ht hint.Hint) (Hal, error) {
	switch ht.Type() {
	case currency.CreateAccountsType:
		return bl.templateCreateAccountsFact(), nil
	case currency.KeyUpdaterType:
		return bl.templateKeyUpdaterFact(), nil
	case currency.TransfersType:
		return bl.templateTransfersFact(), nil
	case currency.CurrencyRegisterType:
		return bl.templateCurrencyRegisterFact(), nil
	case currency.CurrencyPolicyUpdaterType:
		return bl.templateCurrencyPolicyUpdaterFact(), nil
	default:
		return nil, errors.Errorf("unknown operation, %q", ht)
	}
}

func (Builder) templateCreateAccountsFact() Hal {
	nkey, _ := currency.NewBaseAccountKey(templatePublickey, 100)
	nkeys, _ := currency.NewBaseAccountKeys([]currency.AccountKey{nkey}, 100)

	fact := currency.NewCreateAccountsFact(
		templateToken,
		templateSender,
		[]currency.CreateAccountsItem{currency.NewCreateAccountsItemSingleAmount(
			nkeys,
			currency.NewAmount(templateBig, templateCurrencyID),
		)},
	)

	hal := NewBaseHal(fact, HalLink{})
	return hal.AddExtras("default", map[string]interface{}{
		"token":               templateToken,
		"sender":              templateSender,
		"items.keys.keys.key": templatePublickey,
		"items.big":           templateBig,
		"currency":            templateCurrencyID,
	})
}

func (Builder) templateKeyUpdaterFact() Hal {
	nkey, _ := currency.NewBaseAccountKey(templatePublickey, 100)
	nkeys, _ := currency.NewBaseAccountKeys([]currency.AccountKey{nkey}, 100)

	fact := currency.NewKeyUpdaterFact(
		templateToken,
		templateSender,
		nkeys,
		templateCurrencyID,
	)

	hal := NewBaseHal(fact, HalLink{})
	return hal.AddExtras("default", map[string]interface{}{
		"token":         templateToken,
		"target":        templateSender,
		"keys.keys.key": templatePublickey,
		"currency":      templateCurrencyID,
	})
}

func (Builder) templateTransfersFact() Hal {
	fact := currency.NewTransfersFact(
		templateToken,
		templateSender,
		[]currency.TransfersItem{currency.NewTransfersItemSingleAmount(
			templateReceiver,
			currency.NewAmount(templateBig, templateCurrencyID),
		)},
	)

	hal := NewBaseHal(fact, HalLink{})

	return hal.AddExtras("default", map[string]interface{}{
		"token":          templateToken,
		"sender":         templateSender,
		"items.receiver": templateReceiver,
		"items.big":      templateBig,
		"items.currency": templateCurrencyID,
	})
}

func (Builder) templateCurrencyRegisterFact() Hal {
	po := currency.NewCurrencyPolicy(templateBig, currency.NewNilFeeer())
	de := currency.NewCurrencyDesign(
		currency.NewAmount(templateBig, templateCurrencyID),
		templateReceiver,
		po,
	)
	fact := currency.NewCurrencyRegisterFact(templateToken, de)

	hal := NewBaseHal(fact, HalLink{})

	return hal.AddExtras("default", map[string]interface{}{
		"token":                    templateToken,
		"amount.amount":            templateBig,
		"amount.currency":          templateCurrencyID,
		"currency.genesis_account": templateReceiver,
		"currency.policy.new_account_min_balance": templateBig,
	})
}

func (Builder) templateCurrencyPolicyUpdaterFact() Hal {
	po := currency.NewCurrencyPolicy(templateBig, currency.NewNilFeeer())
	fact := currency.NewCurrencyPolicyUpdaterFact(templateToken, templateCurrencyID, po)

	hal := NewBaseHal(fact, HalLink{})

	return hal.AddExtras("default", map[string]interface{}{
		"token":                          templateToken,
		"currency":                       templateCurrencyID,
		"policy.new_account_min_balance": templateBig,
	})
}

func (bl Builder) BuildFact(b []byte) (Hal, error) {
	var fact base.Fact
	if hinter, err := bl.enc.Decode(b); err != nil {
		return nil, err
	} else if f, ok := hinter.(base.Fact); !ok {
		return nil, errors.Errorf("not base.Fact, %T", hinter)
	} else {
		fact = f
	}

	switch t := fact.(type) {
	case currency.CreateAccountsFact:
		return bl.buildFactCreateAccounts(t)
	case currency.KeyUpdaterFact:
		return bl.buildFactKeyUpdater(t)
	case currency.TransfersFact:
		return bl.buildFactTransfers(t)
	case currency.CurrencyRegisterFact:
		return bl.buildFactCurrencyRegister(t)
	case currency.CurrencyPolicyUpdaterFact:
		return bl.buildFactCurrencyPolicyUpdater(t)
	default:
		return nil, errors.Errorf("unknown fact, %T", fact)
	}
}

func (bl Builder) buildFactCreateAccounts(fact currency.CreateAccountsFact) (Hal, error) {
	token, err := bl.checkToken(fact.Token())
	if err != nil {
		return nil, err
	}

	items := make([]currency.CreateAccountsItem, len(fact.Items()))
	for i := range fact.Items() {
		item := fact.Items()[i]
		if len(item.Amounts()) < 1 {
			return nil, errors.Errorf("empty Amounts")
		}

		ks, e := currency.NewBaseAccountKeys(item.Keys().Keys(), item.Keys().Threshold())
		if e != nil {
			return nil, e
		}
		items[i] = currency.NewCreateAccountsItemSingleAmount(ks, item.Amounts()[0])
	}

	nfact := currency.NewCreateAccountsFact(token, fact.Sender(), items)
	nfact = nfact.Rebuild()
	if err = bl.isValidFactCreateAccounts(nfact); err != nil {
		return nil, err
	}

	var hal Hal
	hal = NewBaseHal(nil, HalLink{})
	op, err := currency.NewCreateAccounts(
		nfact,
		[]base.FactSign{
			base.RawBaseFactSign(templatePublickey, templateSignature, templateSignedAt),
		},
		"",
	)
	if err != nil {
		return nil, err
	}
	hal = hal.SetInterface(op)

	return hal.
		AddExtras("default", map[string]interface{}{
			"fact_signs.signer":    templatePublickey,
			"fact_signs.signature": templateSignature,
		}).
		AddExtras("signature_base", base.NewBytesForFactSignature(nfact, bl.networkID)), nil
}

func (bl Builder) buildFactKeyUpdater(fact currency.KeyUpdaterFact) (Hal, error) {
	token, err := bl.checkToken(fact.Token())
	if err != nil {
		return nil, err
	}

	ks, err := currency.NewBaseAccountKeys(fact.Keys().Keys(), fact.Keys().Threshold())
	if err != nil {
		return nil, err
	}

	nfact := currency.NewKeyUpdaterFact(token, fact.Target(), ks, fact.Currency())
	if err = bl.isValidFactKeyUpdater(nfact); err != nil {
		return nil, err
	}

	var hal Hal
	hal = NewBaseHal(nil, HalLink{})
	op, err := currency.NewKeyUpdater(
		nfact,
		[]base.FactSign{
			base.RawBaseFactSign(templatePublickey, templateSignature, templateSignedAt),
		},
		"",
	)
	if err != nil {
		return nil, err
	}
	hal = hal.SetInterface(op)

	return hal.
		AddExtras("default", map[string]interface{}{
			"fact_signs.signer":    templatePublickey,
			"fact_signs.signature": templateSignature,
		}).
		AddExtras("signature_base", base.NewBytesForFactSignature(nfact, bl.networkID)), nil
}

func (bl Builder) buildFactTransfers(fact currency.TransfersFact) (Hal, error) {
	token, err := bl.checkToken(fact.Token())
	if err != nil {
		return nil, err
	}

	nfact := currency.NewTransfersFact(token, fact.Sender(), fact.Items())
	nfact = nfact.Rebuild()
	if err = bl.isValidFactTransfers(nfact); err != nil {
		return nil, err
	}

	var hal Hal
	hal = NewBaseHal(nil, HalLink{})
	op, err := currency.NewTransfers(
		nfact,
		[]base.FactSign{
			base.RawBaseFactSign(templatePublickey, templateSignature, templateSignedAt),
		},
		"",
	)
	if err != nil {
		return nil, err
	}
	hal = hal.SetInterface(op)

	return hal.
		AddExtras("default", map[string]interface{}{
			"fact_signs.signer":    templatePublickey,
			"fact_signs.signature": templateSignature,
		}).
		AddExtras("signature_base", base.NewBytesForFactSignature(nfact, bl.networkID)), nil
}

func (bl Builder) buildFactCurrencyRegister(fact currency.CurrencyRegisterFact) (Hal, error) {
	token, err := bl.checkToken(fact.Token())
	if err != nil {
		return nil, err
	}

	nfact := currency.NewCurrencyRegisterFact(token, fact.Currency())
	if err = bl.isValidFactCurrencyRegister(nfact); err != nil {
		return nil, err
	}

	var hal Hal
	hal = NewBaseHal(nil, HalLink{})
	op, err := currency.NewCurrencyRegister(
		nfact,
		[]base.FactSign{
			base.RawBaseFactSign(templatePublickey, templateSignature, templateSignedAt),
		},
		"",
	)
	if err != nil {
		return nil, err
	}
	hal = hal.SetInterface(op)

	return hal.
		AddExtras("default", map[string]interface{}{
			"fact_signs.signer":    templatePublickey,
			"fact_signs.signature": templateSignature,
		}).
		AddExtras("signature_base", base.NewBytesForFactSignature(nfact, bl.networkID)), nil
}

func (bl Builder) buildFactCurrencyPolicyUpdater(fact currency.CurrencyPolicyUpdaterFact) (Hal, error) {
	token, err := bl.checkToken(fact.Token())
	if err != nil {
		return nil, err
	}

	nfact := currency.NewCurrencyPolicyUpdaterFact(token, fact.Currency(), fact.Policy())
	if err = bl.isValidFactCurrencyPolicyUpdater(nfact); err != nil {
		return nil, err
	}

	var hal Hal
	hal = NewBaseHal(nil, HalLink{})
	op, err := currency.NewCurrencyPolicyUpdater(
		nfact,
		[]base.FactSign{
			base.RawBaseFactSign(templatePublickey, templateSignature, templateSignedAt),
		},
		"",
	)
	if err != nil {
		return nil, err
	}
	hal = hal.SetInterface(op)

	return hal.
		AddExtras("default", map[string]interface{}{
			"fact_signs.signer":    templatePublickey,
			"fact_signs.signature": templateSignature,
		}).
		AddExtras("signature_base", base.NewBytesForFactSignature(nfact, bl.networkID)), nil
}

func (Builder) isValidFactCreateAccounts(fact currency.CreateAccountsFact) error {
	if err := fact.IsValid(nil); err != nil {
		return err
	}

	if bytes.Equal(fact.Token(), templateToken) {
		return errors.Errorf("Please set token; token same with template default")
	}

	if fact.Sender().Equal(templateSender) {
		return errors.Errorf("Please set sender; sender is same with template default")
	}

	for i := range fact.Items() {
		if _, same := fact.Items()[i].Keys().Key(templatePublickey); same {
			return errors.Errorf("Please set key; key is same with template default")
		}
	}

	return nil
}

func (Builder) isValidFactKeyUpdater(fact currency.KeyUpdaterFact) error {
	if err := fact.IsValid(nil); err != nil {
		return err
	}

	if bytes.Equal(fact.Token(), templateToken) {
		return errors.Errorf("Please set token; token same with template default")
	}

	if fact.Target().Equal(templateSender) {
		return errors.Errorf("Please set target; target is same with template default")
	}

	if _, same := fact.Keys().Key(templatePublickey); same {
		return errors.Errorf("Please set key; key is same with template default")
	}

	return nil
}

func (Builder) isValidFactTransfers(fact currency.TransfersFact) error {
	if err := fact.IsValid(nil); err != nil {
		return err
	}

	if bytes.Equal(fact.Token(), templateToken) {
		return errors.Errorf("Please set token; token same with template default")
	}

	if fact.Sender().Equal(templateSender) {
		return errors.Errorf("Please set sender; sender is same with template default")
	}

	for i := range fact.Items() {
		if fact.Items()[i].Receiver().Equal(templateReceiver) {
			return errors.Errorf("Please set receiver; receiver is same with template default")
		}
	}

	return nil
}

func (Builder) isValidFactCurrencyRegister(fact currency.CurrencyRegisterFact) error {
	if err := fact.IsValid(nil); err != nil {
		return err
	}

	if bytes.Equal(fact.Token(), templateToken) {
		return errors.Errorf("Please set token; token same with template default")
	}

	if fact.Currency().GenesisAccount().Equal(templateReceiver) {
		return errors.Errorf("Please set genesis_account; genesis_account is same with template default")
	}

	if fact.Currency().Policy().NewAccountMinBalance().Equal(templateBig) {
		return errors.Errorf("Please set new_account_min_balance; new_account_min_balance is same with template default")
	}

	return nil
}

func (Builder) isValidFactCurrencyPolicyUpdater(fact currency.CurrencyPolicyUpdaterFact) error {
	if err := fact.IsValid(nil); err != nil {
		return err
	}

	if bytes.Equal(fact.Token(), templateToken) {
		return errors.Errorf("Please set token; token same with template default")
	}

	return nil
}

func (bl Builder) BuildOperation(b []byte) (Hal, error) {
	var op operation.Operation
	if hinter, err := bl.enc.Decode(b); err != nil {
		return nil, err
	} else if f, ok := hinter.(operation.Operation); !ok {
		return nil, errors.Errorf("not operation.Operation, %T", hinter)
	} else {
		op = f
	}

	var hal Hal
	if err := func() error {
		var err error
		switch t := op.(type) {
		case currency.CreateAccounts:
			hal, err = bl.buildCreateAccounts(t)
		case currency.KeyUpdater:
			hal, err = bl.buildKeyUpdater(t)
		case currency.Transfers:
			hal, err = bl.buildTransfers(t)
		case currency.CurrencyRegister:
			hal, err = bl.buildCurrencyRegister(t)
		case currency.CurrencyPolicyUpdater:
			hal, err = bl.buildCurrencyPolicyUpdater(t)
		default:
			return errors.Errorf("unknown operation.Operation, %T", t)
		}

		return err
	}(); err != nil {
		return nil, err
	}

	nop := hal.Interface().(operation.Operation)
	for i := range nop.Signs() {
		fs := nop.Signs()[i]
		if fs.Signer().Equal(templatePublickey) {
			return nil, errors.Errorf("Please set publickey; signer is same with template default")
		}

		if fs.Signature().Equal(templateSignature) {
			return nil, errors.Errorf("Please set signature; signature same with template default")
		}
	}

	return hal, nil
}

func (bl Builder) buildCreateAccounts(op currency.CreateAccounts) (Hal, error) {
	fs := bl.updateFactSigns(op.Signs())

	if nop, err := currency.NewCreateAccounts(op.Fact().(currency.CreateAccountsFact), fs, op.Memo); err != nil {
		return nil, err
	} else if err := nop.IsValid(bl.networkID); err != nil {
		return nil, err
	} else if err := bl.isValidFactCreateAccounts(nop.Fact().(currency.CreateAccountsFact)); err != nil {
		return nil, err
	} else {
		return NewBaseHal(nop, HalLink{}), nil
	}
}

func (bl Builder) buildKeyUpdater(op currency.KeyUpdater) (Hal, error) {
	fs := bl.updateFactSigns(op.Signs())

	if nop, err := currency.NewKeyUpdater(op.Fact().(currency.KeyUpdaterFact), fs, op.Memo); err != nil {
		return nil, err
	} else if err := nop.IsValid(bl.networkID); err != nil {
		return nil, err
	} else if err := bl.isValidFactKeyUpdater(nop.Fact().(currency.KeyUpdaterFact)); err != nil {
		return nil, err
	} else {
		return NewBaseHal(nop, HalLink{}), nil
	}
}

func (bl Builder) buildTransfers(op currency.Transfers) (Hal, error) {
	fs := bl.updateFactSigns(op.Signs())

	if nop, err := currency.NewTransfers(op.Fact().(currency.TransfersFact), fs, op.Memo); err != nil {
		return nil, err
	} else if err := nop.IsValid(bl.networkID); err != nil {
		return nil, err
	} else if err := bl.isValidFactTransfers(nop.Fact().(currency.TransfersFact)); err != nil {
		return nil, err
	} else {
		return NewBaseHal(nop, HalLink{}), nil
	}
}

func (bl Builder) buildCurrencyRegister(op currency.CurrencyRegister) (Hal, error) {
	fs := bl.updateFactSigns(op.Signs())

	if nop, err := currency.NewCurrencyRegister(op.Fact().(currency.CurrencyRegisterFact), fs, op.Memo); err != nil {
		return nil, err
	} else if err := nop.IsValid(bl.networkID); err != nil {
		return nil, err
	} else if err := bl.isValidFactCurrencyRegister(nop.Fact().(currency.CurrencyRegisterFact)); err != nil {
		return nil, err
	} else {
		return NewBaseHal(nop, HalLink{}), nil
	}
}

func (bl Builder) buildCurrencyPolicyUpdater(op currency.CurrencyPolicyUpdater) (Hal, error) {
	fs := bl.updateFactSigns(op.Signs())

	if nop, err := currency.NewCurrencyPolicyUpdater(
		op.Fact().(currency.CurrencyPolicyUpdaterFact),
		fs,
		op.Memo,
	); err != nil {
		return nil, err
	} else if err := nop.IsValid(bl.networkID); err != nil {
		return nil, err
	} else if err := bl.isValidFactCurrencyPolicyUpdater(nop.Fact().(currency.CurrencyPolicyUpdaterFact)); err != nil {
		return nil, err
	} else {
		return NewBaseHal(nop, HalLink{}), nil
	}
}

// checkToken checks token is valid; empty token will be updated with current
// time.
func (Builder) checkToken(token []byte) ([]byte, error) {
	if len(token) < 1 {
		return nil, errors.Errorf("empty token")
	}

	if bytes.Equal(token, templateToken) {
		return localtime.NewTime(localtime.UTCNow()).Bytes(), nil
	}

	return token, nil
}

// updateFactSigns regenerate the newly added factsign.
func (Builder) updateFactSigns(fss []base.FactSign) []base.FactSign {
	ufss := make([]base.FactSign, len(fss))
	for i := range fss {
		fs := fss[i]

		if localtime.RFC3339(fs.SignedAt()) == localtime.RFC3339(templateSignedAt) {
			fs = base.NewBaseFactSign(fs.Signer(), fs.Signature())
		}

		ufss[i] = fs
	}

	return ufss
}
