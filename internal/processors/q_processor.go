package processors

import (
	"log/slog"
	"ob-manager/internal/dtos"
	inqueues "ob-manager/internal/queues/in"
	outqueues "ob-manager/internal/queues/out"
	"strconv"
)

type Processor struct {
	inQ  *inqueues.InQManager
	outQ *outqueues.Queue

	currency string
	isReady  chan bool
	ob       *OrderBook
	quit     chan struct{}
}

func NewProcessor(currency string, inQ *inqueues.InQManager, outQ *outqueues.Queue) *Processor {
	return &Processor{
		currency: currency,
		inQ:      inQ,
		outQ:     outQ,
		isReady:  make(chan bool),
		quit:     make(chan struct{}),
		ob:       NewOrderBook(),
	}
}

func (p *Processor) OrderBook() *OrderBook {
	return p.ob
}

func (p *Processor) SetReady(lastUpdateId int) {
	p.ob.SetLastUpdateId(lastUpdateId)
	p.isReady <- true
}

func (p *Processor) startProcessor() {
	<-p.isReady

	for {
		select {
		case <-p.quit:
			slog.Info("Processor Quitting.", "Currency", p.currency)

			return
		case event := <-p.inQ.Queue(p.currency):
			// discard unnecessary bids/asks
			if event.FinalUpdateId < p.ob.LastUpdateId() {
				// discard
				slog.Info("Discarding event.", "curr", p.currency, "Final Id", event.FinalUpdateId, "Last Id", p.ob.LastUpdateId())

				continue
			}

			slog.Debug("Processing event.", "curr", p.currency, "Final Id", event.FinalUpdateId, "Last Id", p.ob.LastUpdateId())

			// process event
			p.updateOrderBook(event)

			// push update to users
			p.outQ.AddToOutQ(event)
		}
	}
}

func (p *Processor) updateOrderBook(event *dtos.EventUpdate) {
	bids := p.processEventBids(event.Bids)
	asks := p.processEventAsks(event.Asks)

	p.ob.lastUpdateId = event.FinalUpdateId
	p.ob.batchUpdate(bids, asks, event.FinalUpdateId)
}

// process bids and populate the order book.
func (p *Processor) processEventBids(bids [][]string) map[float64]float64 {
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

	return bidsMap
}

// process asks and populate the order book.
func (p *Processor) processEventAsks(asks [][]string) map[float64]float64 {
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

	return asksMap
}

func (p *Processor) stopProcessor() {
	close(p.quit)
}
