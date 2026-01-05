package binance

import (
	"fmt"
	"log"
	"log/slog"
	"ob-manager/internal/binance/dtos"
	"strconv"
	"time"
)

type BinanceProcessor struct {
	feed *BinanceFeed
}

func NewBinanceProcessor() *BinanceProcessor {
	return &BinanceProcessor{}
}

// process only market depth updates
func (processor *BinanceProcessor) processMarketDepthUpdate(eventUpdate dtos.EventUpdate, message []byte) {
	if eventUpdate.Symbol == "" {
		// not a market depth update message
		return
	}

	if processor.feed.firstEntryMap[eventUpdate.Symbol] {
		firstUpdateId := eventUpdate.FirstUpdateId
		fmt.Printf("first update Id %d for currency %s \n ", firstUpdateId, eventUpdate.Symbol)
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
	fmt.Println("admin message received: ", string(message))
	if subscriptionsList.Id == processor.feed.listSubscReqId {
		fmt.Println("sending subs list to channel ", subscriptionsList.Result)
		processor.feed.listSubscriptions <- subscriptionsList.Result
	}
}

// event processor to process market depth updates
func (processor *BinanceProcessor) updateEvents() {
	fmt.Println("Event Processor Started")
	for {
		eventUpdate := <-processor.feed.bufferedEvents
		currPair := eventUpdate.Symbol

		if eventUpdate.FinalUpdateId <= processor.feed.lastUpdateIds[currPair] {
			// not an eligible event to process
			continue
		}

		if eventUpdate.EventType != depthUpdateEvent {
			// Order Book Manager do not need to process these events
			fmt.Printf("Event type for curr pair %s %s \n", currPair, eventUpdate.EventType)
			continue
		}

		formattedText := fmt.Sprintf("processing event for curr pair: %s %d %d %s", currPair, eventUpdate.FirstUpdateId, eventUpdate.FinalUpdateId, time.Now())
		fmt.Println(formattedText)

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
		priceVal, err := strconv.ParseFloat(bidEntry[0], 64)
		if err != nil {
			log.Printf("Error on Parsing Bid Entry Price for curr pair %s \n", currPair)
		}

		qtyVal, err := strconv.ParseFloat(bidEntry[1], 64)
		if err != nil {
			log.Printf("Error on Parsing Bid Entry Quantity for curr pair %s \n", currPair)
		}

		slog.Info("bid processed", "price: ", priceVal, " qty: ", qtyVal)
		// orderbook.UpdateBids(currPair, priceVal, qtyVal)
	}
}

// process asks and update order book from snapshot
func (processor *BinanceProcessor) processAsks(asks [][]string, currPair string) {
	for _, askEntry := range asks {
		priceVal, err := strconv.ParseFloat(askEntry[0], 64)
		if err != nil {
			log.Printf("Error on Parsing Ask Entry Price for curr pair %s \n", currPair)
		}

		qtyVal, err := strconv.ParseFloat(askEntry[1], 64)
		if err != nil {
			log.Printf("Error on Parsing Ask Entry Quantity for curr pair %s \n", currPair)
		}

		slog.Info("bid processed", "price: ", priceVal, " qty: ", qtyVal)
		// orderbook.UpdateAsks(currPair, priceVal, qtyVal)
	}
}
