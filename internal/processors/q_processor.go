package processors

import (
	"log/slog"
	"strconv"
)

type Processor struct {
	DeQueuer
	OutQ

	currency     string
	lastUpdateId int
	isReady      chan bool
	ob           *OrderBook
}

func NewProcessor(currency string, outQ OutQ, deQueuer DeQueuer) *Processor {
	return &Processor{
		currency: currency,
		DeQueuer: deQueuer,
		OutQ:     outQ,
		isReady:  make(chan bool),
		ob:       NewOrderBook(),
	}
}

func (p *Processor) OrderBook() *OrderBook {
	return p.ob
}

func (p *Processor) SetReady(lastUpdateId int) {
	p.lastUpdateId = lastUpdateId
	p.isReady <- true
}

func (p *Processor) startProcessor() {
	<-p.isReady

	for {
		event := <-p.DeQueue(p.currency)

		// discard unnecessary bids/asks
		if event.FinalUpdateId < p.lastUpdateId {
			// discard
			slog.Info("Discarding event.", "curr", p.currency, "Final Id", event.FinalUpdateId, "Last Id", p.lastUpdateId)

			continue
		}

		slog.Debug("Processing event.", "curr", p.currency, "Final Id", event.FinalUpdateId, "Last Id", p.lastUpdateId)

		//process event
		p.processEventBids(event.Bids)
		p.processEventAsks(event.Asks)

		p.lastUpdateId = event.FinalUpdateId

		// push update to users
		p.AddToOutQ(event)
	}
}

// process bids and populate order book.
func (p *Processor) processEventBids(bids [][]string) {
	bidsMap := make(map[float64]float64)

	for _, bidEntry := range bids {
		price, err := strconv.ParseFloat(bidEntry[0], 64)
		if err != nil {
			slog.Error("Error on Parsing Bid Entry Price", "Error", err)
		}

		qty, err := strconv.ParseFloat(bidEntry[1], 64)
		if err != nil {
			slog.Error("Error on Parsing Bid Entry Quantity", "Error", err)
		}

		bidsMap[price] = qty
	}

	p.updateBids(bidsMap)
}

// process asks and populate order book.
func (p *Processor) processEventAsks(asks [][]string) {
	asksMap := make(map[float64]float64)

	for _, askEntry := range asks {
		price, err := strconv.ParseFloat(askEntry[0], 64)
		if err != nil {
			slog.Error("Error on Parsing Ask Entry Price", "Error", err)
		}

		qty, err := strconv.ParseFloat(askEntry[1], 64)
		if err != nil {
			slog.Error("Error on Parsing Bid Entry Quantity", "Error", err)
		}

		asksMap[price] = qty
	}

	p.updateBids(asksMap)
}

func (p *Processor) updateBids(bids map[float64]float64) {
	for price, qty := range bids {
		p.ob.Bids.Put(price, qty)
	}
}

func (p *Processor) updateAsks(asks map[float64]float64) {
	for price, qty := range asks {
		p.ob.Asks.Put(price, qty)
	}
}
