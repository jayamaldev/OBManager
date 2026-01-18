package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"ob-manager/internal/dtos"
	"strconv"
)

type RestClient struct {
	ProcManager
}

func NewRestClient(updater ProcManager) *RestClient {
	return &RestClient{
		ProcManager: updater,
	}
}

// GetSnapshot to get the market depth for a currency pair and populate the order book.
func (c *RestClient) GetSnapshot(ctx context.Context, currPair string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(snapshotURL, currPair), nil)
	if err != nil {
		slog.Error("Error on Creating New GET Request", "curr pair", currPair, "Error", err)

		return err
	}

	client := &http.Client{}

	slog.Info("Sending Rest Request to get Market Depth", "Currency", currPair)

	resp, err := client.Do(req)

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			slog.Error("Error on Closing Response", "curr pair", currPair, "Error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		slog.Error("Error on Getting Snapshot", "curr pair", currPair, "Error", err)

		return err
	}

	var snapshot *dtos.Snapshot
	err = json.NewDecoder(resp.Body).Decode(&snapshot)

	if err != nil {
		slog.Error("Error on parse Response Json", "curr pair", currPair, "Error", err)

		return err
	}

	c.updateSnapshot(currPair, snapshot)

	return nil
}

func (c *RestClient) updateSnapshot(currPair string, snapshot *dtos.Snapshot) {
	// process bids
	c.processBids(currPair, snapshot.Bids)

	// process asks
	c.processAsks(currPair, snapshot.Asks)

	// flag snapshot populated. start consuming push events.
	c.SetOrderBookReady(currPair, snapshot.LastUpdateId)
}

// process bids and populate order book.
func (c *RestClient) processBids(currPair string, bids [][]string) {
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

	c.UpdateBids(currPair, bidsMap)
}

// process asks and populate order book.
func (c *RestClient) processAsks(currPair string, asks [][]string) {
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

	c.UpdateAsks(currPair, asksMap)
}
