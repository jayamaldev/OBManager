package processors

import (
	tree "github.com/emirpasic/gods/trees/redblacktree"
)

type OrderBook struct {
	Bids         *tree.Tree
	Asks         *tree.Tree
	lastUpdateId int
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		Bids: tree.NewWith(bidComparator),
		Asks: tree.NewWith(askComparator),
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
