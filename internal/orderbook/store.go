package orderbook

import (
	"encoding/json"
	"log/slog"
	"sync"

	tree "github.com/emirpasic/gods/trees/redblacktree"
)

type OrderBook struct {
	Bids *tree.Tree
	Asks *tree.Tree
}

type OBStore struct {
	mu    sync.RWMutex
	store map[string]*OrderBook
}

func NewStore() *OBStore {
	return &OBStore{
		store: make(map[string]*OrderBook),
	}
}

// initialize order book instance from the ob store for the given currency
func (s *OBStore) InitOrderBook(currency string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.store[currency] = &OrderBook{
		Bids: tree.NewWith(bidComparator),
		Asks: tree.NewWith(askComparator),
	}
	slog.Info("Order Book Initiated for ", "currency", currency)
}

// remove order book instance from the ob store for a given currenct
func (s *OBStore) RemoveOrderBook(currency string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.store, currency)
}

// update bids of a order book
func (s *OBStore) UpdateBids(currency string, bids map[float64]float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for price, qty := range bids {
		s.store[currency].Bids.Put(price, qty)
	}
}

// update asks of a order book
func (s *OBStore) UpdateAsks(currency string, asks map[float64]float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for price, qty := range asks {
		s.store[currency].Bids.Put(price, qty)
	}
}

// return json string of a order book to send to the subscribed user
func (s *OBStore) GetOrderBook(currency string) []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	orderBook := s.store[currency]
	jsonStr, err := json.Marshal(orderBook)
	if err != nil {
		slog.Error("error on parsing order book to json", "Error", err)
	}

	return jsonStr
}

// comparator to sort asks
func askComparator(a, b interface{}) int {
	aFloat := a.(float64)
	bFloat := b.(float64)

	if aFloat < bFloat {
		return -1
	}
	if aFloat > bFloat {
		return 1
	}
	return 0
}

// comparator to sort bids
func bidComparator(a, b interface{}) int {
	aFloat := a.(float64)
	bFloat := b.(float64)

	if aFloat < bFloat {
		return 1
	}
	if aFloat > bFloat {
		return -1
	}
	return 0
}
