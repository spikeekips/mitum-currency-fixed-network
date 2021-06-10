package currency

import (
	"sync"

	"github.com/spikeekips/mitum/base/state"
)

type CurrencyPool struct {
	sync.RWMutex
	demap  map[CurrencyID]CurrencyDesign
	stsmap map[CurrencyID]state.State
	cids   []CurrencyID
}

func NewCurrencyPool() *CurrencyPool {
	return &CurrencyPool{
		demap:  map[CurrencyID]CurrencyDesign{},
		stsmap: map[CurrencyID]state.State{},
	}
}

func (cp *CurrencyPool) Clear() {
	cp.Lock()
	defer cp.Unlock()

	cp.demap = nil
	cp.stsmap = nil
	cp.cids = nil
}

func (cp *CurrencyPool) Set(st state.State) error {
	cp.Lock()
	defer cp.Unlock()

	de, err := StateCurrencyDesignValue(st)
	if err != nil {
		return err
	}

	cp.demap[de.Currency()] = de
	cp.stsmap[de.Currency()] = st
	cp.cids = append(cp.cids, de.Currency())

	return nil
}

func (cp *CurrencyPool) CIDs() []CurrencyID {
	cp.RLock()
	defer cp.RUnlock()

	return cp.cids
}

func (cp *CurrencyPool) Designs() map[CurrencyID]CurrencyDesign {
	cp.RLock()
	defer cp.RUnlock()

	m := map[CurrencyID]CurrencyDesign{}
	for k := range cp.demap {
		m[k] = cp.demap[k]
	}

	return m
}

func (cp *CurrencyPool) States() map[CurrencyID]state.State {
	cp.RLock()
	defer cp.RUnlock()

	m := map[CurrencyID]state.State{}
	for k := range cp.stsmap {
		m[k] = cp.stsmap[k]
	}

	return m
}

func (cp *CurrencyPool) Policy(cid CurrencyID) (CurrencyPolicy, bool) {
	i, found := cp.Get(cid)
	if !found {
		return CurrencyPolicy{}, false
	}
	return i.Policy(), true
}

func (cp *CurrencyPool) Feeer(cid CurrencyID) (Feeer, bool) {
	i, found := cp.Get(cid)
	if !found {
		return nil, false
	}
	return i.Policy().Feeer(), true
}

func (cp *CurrencyPool) State(cid CurrencyID) (state.State, bool) {
	i, found := cp.stsmap[cid]
	return i, found
}

func (cp *CurrencyPool) TraverseDesign(callback func(cid CurrencyID, de CurrencyDesign) bool) {
	cp.RLock()
	defer cp.RUnlock()

	for k := range cp.demap {
		if !callback(k, cp.demap[k]) {
			break
		}
	}
}

func (cp *CurrencyPool) TraverseState(callback func(cid CurrencyID, de state.State) bool) {
	cp.RLock()
	defer cp.RUnlock()

	for k := range cp.stsmap {
		if !callback(k, cp.stsmap[k]) {
			break
		}
	}
}

func (cp *CurrencyPool) Exists(cid CurrencyID) bool {
	cp.RLock()
	defer cp.RUnlock()

	_, found := cp.demap[cid]

	return found
}

func (cp *CurrencyPool) Get(cid CurrencyID) (CurrencyDesign, bool) {
	cp.RLock()
	defer cp.RUnlock()

	i, found := cp.demap[cid]
	return i, found
}
