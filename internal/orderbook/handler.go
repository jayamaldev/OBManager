package orderbook

// provides abstration of the orderbook to the application
type Store interface {
	InitOrderBook(currency string)
	RemoveOrderBook(currency string)
	UpdateBids(currency string, bids map[float64]float64)
	UpdateAsks(currency string, asks map[float64]float64)
	GetOrderBook(currency string) []byte
}

type StoreHandler struct {
	Store
}

func NewStoreHandler(store Store) *StoreHandler {
	return &StoreHandler{
		Store: store,
	}
}
