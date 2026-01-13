package upstream

import (
	"context"
	"ob-manager/internal/subscribers"
	"ob-manager/internal/upstream/binance"

	"golang.org/x/sync/errgroup"
)

type Client struct {
	upstream     *Upstream
	mainCurrPair string
	storeUpdater *binance.OBUpdater
}

// wire everything on upstream and return the client struct
func NewClient(handler binance.Updater, subHandler *subscribers.Handler) *Client {
	// make required channels
	updateIdChan := make(chan int)
	listSubscriptions := make(chan []string)
	requests := make(chan []byte)

	binanceRegistry := binance.NewMarketDepthRegistry()
	storeUpdater := binance.NewOBUpdater(handler)

	binanceWS := binance.NewWSClient(requests)
	binanceProcessor := binance.NewProcessor(updateIdChan, listSubscriptions, binanceRegistry, storeUpdater, subHandler)
	binanceSubscriber := binance.NewSubscriber(updateIdChan, listSubscriptions, requests, binanceRegistry, storeUpdater)

	upstream := NewUpstream(binanceWS, binanceProcessor, binanceSubscriber)

	return &Client{
		upstream:     upstream,
		storeUpdater: storeUpdater,
	}
}

func (c *Client) InitClient(g *errgroup.Group, ctx context.Context) {
	c.storeUpdater.InitOrderBook(c.mainCurrPair)
	c.upstream.initUpstream(ctx, g, c.mainCurrPair)
}

func (c *Client) SetMainCurrencyPair(currencyPair string) {
	c.mainCurrPair = currencyPair
}

func (c *Client) CloseClient() error {
	return c.upstream.closeUpstream()
}
