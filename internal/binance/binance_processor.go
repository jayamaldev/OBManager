package binance

import (
	"log/slog"
	"ob-manager/internal/binance/dtos"
	"strconv"
	"time"
)

type BinanceProcessor struct {
	feed *BinanceFeed
}

func NewBinanceProcessor(feed *BinanceFeed) *BinanceProcessor {
	return &BinanceProcessor{
		feed: feed,
	}
}

// process only market depth updates
func (processor *BinanceProcessor) processMarketDepthUpdate(eventUpdate dtos.EventUpdate, message []byte) {
	if eventUpdate.Symbol == "" {
		// not a market depth update message
		return
	}

	if processor.feed.firstEntryMap[eventUpdate.Symbol] {
		firstUpdateId := eventUpdate.FirstUpdateId
		slog.Info("Updating Event.", "first update id", firstUpdateId, "currency pair", eventUpdate.Symbol)
		processor.feed.firstEntryMap[eventUpdate.Symbol] = false
		processor.feed.updateIdChan <- firstUpdateId
	}

	// write to bufferedEvents channel so the Event Processor goroutine will read from channel
	processor.feed.bufferedEvents <- eventUpdate

	// push the event to subscribed users
	// users.PushEventToUsers(message, eventUpdate.Symbol)
}

// process list of subscription responses
func (processor *BinanceProcessor) processSubscriptionList(subscriptionsList dtos.SubscriptionsList, message []byte) {
	if subscriptionsList.Id == 0 {
		// not an admin message
		return
	}
	slog.Info("admin message received: ", "message", string(message))
	if subscriptionsList.Id == processor.feed.listSubscReqId {
		slog.Info("sending subs list to channel ", "subs", subscriptionsList.Result)
		processor.feed.listSubscriptions <- subscriptionsList.Result
	}
}

// event processor to process market depth updates
func (processor *BinanceProcessor) updateEvents() {
	slog.Info("Event Processor Started")
	for {
		eventUpdate := <-processor.feed.bufferedEvents
		currPair := eventUpdate.Symbol

		if eventUpdate.FinalUpdateId <= processor.feed.lastUpdateIds[currPair] {
			// not an eligible event to process
			continue
		}

		if eventUpdate.EventType != depthUpdateEvent {
			// Order Book Manager do not need to process these events
			slog.Info("Event type", "curr pair", currPair, "event type", eventUpdate.EventType)
			continue
		}

		slog.Info("processing event", "currency pair", currPair, "first update id", eventUpdate.FirstUpdateId, "last update id", eventUpdate.FinalUpdateId, "time", time.Now())

		// process bids
		processor.processBids(eventUpdate.Bids, currPair)

		// process asks
		processor.processAsks(eventUpdate.Asks, currPair)

		// update last update id of the orderbook
		processor.feed.lastUpdateIds[currPair] = eventUpdate.FinalUpdateId
	}
}

// process bids and update order book from snapshot
func (processor *BinanceProcessor) processBids(bids [][]string, currPair string) {
	for _, bidEntry := range bids {
		_, err := strconv.ParseFloat(bidEntry[0], 64)
		if err != nil {
			slog.Error("Error on Parsing Bid Entry Price", "curr pair", currPair, "Error", err)
		}

		_, err = strconv.ParseFloat(bidEntry[1], 64)
		if err != nil {
			slog.Error("Error on Parsing Bid Entry Quantity", "curr pair", currPair, "Error", err)
		}

		// slog.Info("bid processed", "price: ", priceVal, " qty: ", qtyVal)
		// orderbook.UpdateBids(currPair, priceVal, qtyVal)
	}
}

// process asks and update order book from snapshot
func (processor *BinanceProcessor) processAsks(asks [][]string, currPair string) {
	for _, askEntry := range asks {
		_, err := strconv.ParseFloat(askEntry[0], 64)
		if err != nil {
			slog.Error("Error on Parsing Ask Entry Price", "curr pair", currPair, "Error", err)
		}

		_, err = strconv.ParseFloat(askEntry[1], 64)
		if err != nil {
			slog.Error("Error on Parsing Bid Entry Quantity", "curr pair", currPair, "Error", err)
		}

		// slog.Info("bid processed", "price: ", priceVal, " qty: ", qtyVal)
		// orderbook.UpdateAsks(currPair, priceVal, qtyVal)
	}
}
