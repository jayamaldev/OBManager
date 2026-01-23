package processors

import (
	"ob-manager/internal/dtos"
	"sync"

	tree "github.com/emirpasic/gods/trees/redblacktree"
)

type OrderBook struct {
	Bids         *tree.Tree
	Asks         *tree.Tree
	lastUpdateId int
	mu           sync.RWMutex
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		Bids: tree.NewWith(bidComparator),
		Asks: tree.NewWith(askComparator),
	}
}

func (ob *OrderBook) Snapshot() *dtos.Snapshot {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	bids := make([][]string, 0, ob.Bids.Size())
	bidsIt := ob.Bids.Iterator()

	for bidsIt.Next() {
		bids = append(bids, []string{bidsIt.Key().(string), bidsIt.Value().(string)})
	}

	asks := make([][]string, 0, ob.Asks.Size())
	asksIt := ob.Asks.Iterator()

	for asksIt.Next() {
		asks = append(asks, []string{asksIt.Key().(string), asksIt.Value().(string)})
	}

	return &dtos.Snapshot{
		LastUpdateId: ob.lastUpdateId,
		Bids:         bids,
		Asks:         asks,
	}
}

func (ob *OrderBook) SetLastUpdateId(lastUpdateId int) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	ob.lastUpdateId = lastUpdateId
}

func (ob *OrderBook) LastUpdateId() int {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	return ob.lastUpdateId
}

func (ob *OrderBook) batchUpdate(bids, asks map[float64]float64, lastUpdateId int) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	for price, qty := range bids {
		ob.Bids.Put(price, qty)
	}

	for price, qty := range asks {
		ob.Asks.Put(price, qty)
	}

	ob.lastUpdateId = lastUpdateId
}

func (ob *OrderBook) updateBids(bids map[float64]float64) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	for price, qty := range bids {
		ob.Bids.Put(price, qty)
	}
}

func (ob *OrderBook) updateAsks(asks map[float64]float64) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	for price, qty := range asks {
		ob.Asks.Put(price, qty)
	}
}

// askComparator to sort asks.
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

// bidComparator comparator to sort bids.
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
