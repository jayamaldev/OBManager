package binance

import (
	"encoding/json"
	"log/slog"
	"ob-manager/internal/upstream/binance/dtos"
)

type EventPusher interface {
	PushEventToUsers(message []byte, currency string)
}

type Processor struct {
	EventPusher

	bufferedEvents    chan dtos.EventUpdate
	updateIdChan      chan int
	listSubscriptions chan []string
	mdRegistry        *MarketDepthRegistry
	updater           *OBUpdater
}

// NewProcessor todo too much arguments. change this.
func NewProcessor(updateIdChan chan int, listSubscriptions chan []string, mdRegistry *MarketDepthRegistry, updater *OBUpdater, eventPusher EventPusher) *Processor {
	return &Processor{
		mdRegistry:        mdRegistry,
		bufferedEvents:    make(chan dtos.EventUpdate),
		updateIdChan:      updateIdChan,
		listSubscriptions: listSubscriptions,
		updater:           updater,
		EventPusher:       eventPusher,
	}
}

// ProcessMessage message processor for binance websocket.
func (p *Processor) ProcessMessage(bufferedMsgs chan []byte) error {
	slog.Info("started binance message processor")

	for {
		var (
			eventUpdate       dtos.EventUpdate
			subscriptionsList dtos.SubscriptionsList
		)

		slog.Info("Waiting for messages from binance")

		message := <-bufferedMsgs

		// market depth update
		err := json.Unmarshal(message, &eventUpdate)
		if err != nil {
			slog.Error("Error Parsing Json", "Error", err)

			break
		}

		// subscription list response
		err = json.Unmarshal(message, &subscriptionsList)
		if err != nil {
			slog.Error("Error Parsing Subscriptions List Json", "Error", err)

			break
		}

		// process only market depth updates
		p.processMarketDepthUpdate(eventUpdate, message)

		// process list of subscription responses
		p.processSubscriptionList(subscriptionsList, message)
	}

	return nil
}

// UpdateEvents is the event processor to process market depth updates.
func (p *Processor) UpdateEvents() error {
	slog.Info("Event Processor Started")

	for {
		eventUpdate := <-p.bufferedEvents
		currPair := eventUpdate.Symbol

		if eventUpdate.FinalUpdateId <= p.mdRegistry.LastUpdateId(currPair) {
			// not an eligible event to process
			continue
		}

		if eventUpdate.EventType != depthUpdateEvent {
			// Order Book Manager do not need to process these events
			slog.Info("Event type", "curr pair", currPair, "event type", eventUpdate.EventType)

			continue
		}

		slog.Info("event", "curr pair", currPair, "first id", eventUpdate.FirstUpdateId, "last id", eventUpdate.FinalUpdateId)

		// process bids
		p.updater.processBids(currPair, eventUpdate.Bids)

		// process asks
		p.updater.processAsks(currPair, eventUpdate.Asks)

		// update last update id of the order book
		p.mdRegistry.SetLastUpdateId(currPair, eventUpdate.FinalUpdateId)
	}
}

func (p *Processor) CloseProcessor() {
	close(p.bufferedEvents)
	close(p.listSubscriptions)
	close(p.updateIdChan)
}

// process only market depth updates.
func (p *Processor) processMarketDepthUpdate(eventUpdate dtos.EventUpdate, message []byte) {
	if eventUpdate.Symbol == "" {
		// not a market depth update message
		return
	}

	if p.mdRegistry.FirstEntry(eventUpdate.Symbol) {
		firstUpdateId := eventUpdate.FirstUpdateId
		slog.Info("Updating Event.", "first update id", firstUpdateId, "currency pair", eventUpdate.Symbol)
		p.mdRegistry.SetFirstEntry(eventUpdate.Symbol, false)
		slog.Info("First Entry Updated")
		p.updateIdChan <- firstUpdateId
	}

	slog.Info("adding event to the channel")
	// write to bufferedEvents channel so the Event Processor goroutine will read from channel
	p.bufferedEvents <- eventUpdate

	slog.Info("event added to queue")

	// push the event to subscribed users
	p.PushEventToUsers(message, eventUpdate.Symbol)
}

func (p *Processor) processSubscriptionList(subscriptionsList dtos.SubscriptionsList, message []byte) {
	if subscriptionsList.Id == 0 {
		// not an admin message
		return
	}

	slog.Info("admin message received: ", "message", string(message))

	if subscriptionsList.Id == p.mdRegistry.ListSubsReqId() {
		slog.Info("sending subs list to channel ", "subs", subscriptionsList.Result)
		p.listSubscriptions <- subscriptionsList.Result
	}
}
