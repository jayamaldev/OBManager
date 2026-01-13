package binance

import (
	"sync"
)

type MarketDepthState struct {
	firstEntryMap  map[string]bool
	lastUpdateIds  map[string]int
	listSubscReqId int
}

type MarketDepthRegistry struct {
	state *MarketDepthState
	mutex sync.Mutex
}

func NewMarketDepthRegistry() *MarketDepthRegistry {
	mdState := &MarketDepthState{
		firstEntryMap: make(map[string]bool),
		lastUpdateIds: make(map[string]int),
	}
	return &MarketDepthRegistry{
		state: mdState,
	}
}

func (r *MarketDepthRegistry) SetFirstEntry(currency string, val bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.state.firstEntryMap[currency] = val
}

func (r *MarketDepthRegistry) FirstEntry(currency string) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.state.firstEntryMap[currency]
}

func (r *MarketDepthRegistry) SetLastUpdateId(currency string, id int) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.state.lastUpdateIds[currency] = id
}

func (r *MarketDepthRegistry) LastUpdateId(currency string) int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.state.lastUpdateIds[currency]
}

func (r *MarketDepthRegistry) SetListSubscReqId(reqId int) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.state.listSubscReqId = reqId
}

func (r *MarketDepthRegistry) ListSubscReqId() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.state.listSubscReqId
}
