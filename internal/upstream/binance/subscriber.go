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

// todo too many arguments
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

// Subscription/Unsubscription Request to the Binance to Subscribe for a Currecy Pair
type SubscriptionRequest struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	Id     int      `json:"id"`
}

// Subscription List Request to get the Subscribed Currecy Pairs list from Binance
type ListSubscriptionRequest struct {
	Method string `json:"method"`
	Id     int    `json:"id"`
}

// Subscribe to Currency Pair
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

// get the market depth for a currency pair and populate the order book
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

// UnSubscribe to Currency Pair
func (s *Subscriber) UnsubscribeToCurrPair(currencyPair string) {
	depthRequest := fmt.Sprintf(depthStr, strings.ToLower(currencyPair))
	unsubscriptionRequest := SubscriptionRequest{
		Method: unsubscribe,
		Params: []string{depthRequest},
		Id:     s.uniqueIdGenerator.getUniqueReqId(),
	}

	slog.Info("Unsubscription currency pair", "curr pair", currencyPair)
	s.updater.RemoveOrderBook(currencyPair)

	unsubsRequest, err := json.Marshal(unsubscriptionRequest)
	slog.Info("UnSubscription", "request", string(unsubsRequest))
	if err != nil {
		slog.Error("Error on parsing unsubscription request", "error", err)
	}

	s.requests <- unsubsRequest

	slog.Info("Unsubscribed currency pair", "curr pair", currencyPair)
}

// List of Subscribtions
func (s *Subscriber) ListSubscriptions() []string {
	reqId := s.uniqueIdGenerator.getUniqueReqId()
	s.mdRegistry.SetListSubscReqId(reqId)

	listSubscriptionRequest := ListSubscriptionRequest{
		Method: listSubscriptionsConst,
		Id:     reqId,
	}

	listsubsRequest, err := json.Marshal(listSubscriptionRequest)
	slog.Info("List Subscription Request", "request", string(listsubsRequest))
	if err != nil {
		slog.Error("Error on parsing list subscription request", "error", err)
	}

	s.requests <- listsubsRequest

	subscriptionsList := <-s.listSubscriptions
	return subscriptionsList
}
