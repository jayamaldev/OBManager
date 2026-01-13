package binance

import (
	"log/slog"
	"ob-manager/internal/upstream/binance/dtos"
	"strconv"
)

type Updater interface {
	InitOrderBook(currency string)
	RemoveOrderBook(currency string)
	UpdateBids(currency string, bids map[float64]float64)
	UpdateAsks(currency string, asks map[float64]float64)
}

type OBUpdater struct {
	Updater
}

func NewOBUpdater(u Updater) *OBUpdater {
	return &OBUpdater{
		Updater: u,
	}
}

func (u *OBUpdater) updateSnapshot(currPair string, snapshot *dtos.Snapshot) {
	// process bids
	u.processBids(currPair, snapshot.Bids)

	// process asks
	u.processAsks(currPair, snapshot.Asks)
}

// proceess bids and populate order book
func (u *OBUpdater) processBids(currPair string, bids [][]string) {
	bidsMap := make(map[float64]float64)
	for _, bidEntry := range bids {
		price, err := strconv.ParseFloat(bidEntry[0], 64)
		if err != nil {
			slog.Error("Error on Parsing Bid Entry Price", "curr pair", currPair, "Error", err)
		}

		qty, err := strconv.ParseFloat(bidEntry[1], 64)
		if err != nil {
			slog.Error("Error on Parsing Bid Entry Quantity", "curr pair", currPair, "Error", err)
		}

		bidsMap[price] = qty
	}
	u.UpdateBids(currPair, bidsMap)
}

// proceess asks and populate order book
func (u *OBUpdater) processAsks(currPair string, asks [][]string) {
	asksMap := make(map[float64]float64)
	for _, askEntry := range asks {
		price, err := strconv.ParseFloat(askEntry[0], 64)
		if err != nil {
			slog.Error("Error on Parsing Ask Entry Price", "curr pair", currPair, "Error", err)
		}

		qty, err := strconv.ParseFloat(askEntry[1], 64)
		if err != nil {
			slog.Error("Error on Parsing Bid Entry Quantity", "curr pair", currPair, "Error", err)
		}

		asksMap[price] = qty
	}
	u.UpdateBids(currPair, asksMap)
}
