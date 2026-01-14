package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"ob-manager/internal/upstream/binance/dtos"
	"strings"
)

type Subscriber struct {
	updateIdChan      chan int
	listSubscriptions chan []string
	requests          chan []byte
	uniqueIdGenerator *IDGenerator
	mdRegistry        *MarketDepthRegistry
	updater           *OBUpdater
}

// NewSubscriber todo too many arguments.
func NewSubscriber(updIds chan int, listSubs chan []string, reqs chan []byte, mdRegistry *MarketDepthRegistry, updater *OBUpdater) *Subscriber {
	idGen := NewIDGenerator()

	return &Subscriber{
		updateIdChan:      updIds,
		uniqueIdGenerator: idGen,
		listSubscriptions: listSubs,
		requests:          reqs,
		mdRegistry:        mdRegistry,
		updater:           updater,
	}
}

// SubscriptionRequest to the Binance to Subscribe/ Unsubscribe for a Currency Pair.
type SubscriptionRequest struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	Id     int      `json:"id"`
}

// ListSubscriptionRequest Request to get the Subscribed Currency Pairs list from Binance.
type ListSubscriptionRequest struct {
	Method string `json:"method"`
	Id     int    `json:"id"`
}

func (s *Subscriber) SubscribeToCurrPair(ctx context.Context, currencyPair string) error {
	depthRequest := fmt.Sprintf(depthStr, strings.ToLower(currencyPair))
	subscriptionRequest := SubscriptionRequest{
		Method: subscribe,
		Params: []string{depthRequest},
		Id:     s.uniqueIdGenerator.getUniqueReqId(),
	}

	slog.Info("Subscription currency pair", "curr pair", currencyPair)
	s.updater.InitOrderBook(currencyPair)

	s.mdRegistry.SetFirstEntry(currencyPair, true)

	subsRequest, err := json.Marshal(subscriptionRequest)
	slog.Info("Subscription", "Request", string(subsRequest))

	if err != nil {
		slog.Error("Error on parsing subscription request", "Error", err)
	}

	s.requests <- subsRequest

	return s.getMarketDepth(ctx, currencyPair)
}

func (s *Subscriber) UnsubscribeToCurrPair(currencyPair string) {
	depthRequest := fmt.Sprintf(depthStr, strings.ToLower(currencyPair))
	unsubscriptionRequest := SubscriptionRequest{
		Method: unsubscribe,
		Params: []string{depthRequest},
		Id:     s.uniqueIdGenerator.getUniqueReqId(),
	}

	slog.Info("Unsubscription currency pair", "curr pair", currencyPair)
	s.updater.RemoveOrderBook(currencyPair)

	unSubsRequest, err := json.Marshal(unsubscriptionRequest)
	slog.Info("Unsubscription", "request", string(unSubsRequest))

	if err != nil {
		slog.Error("Error on parsing unsubscription request", "error", err)
	}

	s.requests <- unSubsRequest

	slog.Info("Unsubscribed currency pair", "curr pair", currencyPair)
}

func (s *Subscriber) ListSubscriptions() []string {
	reqId := s.uniqueIdGenerator.getUniqueReqId()
	s.mdRegistry.SetListSubsReqId(reqId)

	listSubscriptionRequest := ListSubscriptionRequest{
		Method: listSubscriptionsConst,
		Id:     reqId,
	}

	listSubsRequest, err := json.Marshal(listSubscriptionRequest)
	slog.Info("List Subscription Request", "request", string(listSubsRequest))

	if err != nil {
		slog.Error("Error on parsing list subscription request", "error", err)
	}

	s.requests <- listSubsRequest

	subscriptionsList := <-s.listSubscriptions

	return subscriptionsList
}

// getMarketDepth to get the market depth for a currency pair and populate the order book.
func (s *Subscriber) getMarketDepth(ctx context.Context, currPair string) error {
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

	lastUpdateId := snapshot.LastUpdateId
	s.mdRegistry.SetLastUpdateId(currPair, lastUpdateId)

	firstUpdateId := <-s.updateIdChan
	slog.Info("last update id", "curr pair", currPair, "last update id", lastUpdateId, "first update id", firstUpdateId)

	if lastUpdateId > firstUpdateId {
		slog.Info("Condition Satisfied", "curr pair", currPair)
	} else {
		slog.Warn("Closing the Application. Re-get snapshot", "curr pair", currPair)

		return err
	}

	s.updater.updateSnapshot(currPair, snapshot)

	return nil
}
