package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"ob-manager/internal/binance/dtos"
	"strings"

	"github.com/gorilla/websocket"
)

type BinanceSubscriber struct {
	feed              *BinanceFeed
	uniqueIdGenerator *IDGenerator
}

func NewBinanceSubscriber(feed *BinanceFeed) *BinanceSubscriber {
	idGen := NewIDGenerator()
	return &BinanceSubscriber{
		uniqueIdGenerator: idGen,
		feed:              feed,
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
func (subscriber *BinanceSubscriber) SubscribeToCurrPair(ctx context.Context, currencyPair string) error {
	depthRequest := fmt.Sprintf(depthStr, strings.ToLower(currencyPair))
	subscriptionRequest := SubscriptionRequest{
		Method: subscribe,
		Params: []string{depthRequest},
		Id:     subscriber.uniqueIdGenerator.getUniqueReqId(),
	}

	slog.Info("Subscription currency pair", "curr pair", currencyPair)
	// orderbook.InitOrderBookForCurrency(currencyPair)
	subscriber.feed.firstEntryMap[currencyPair] = true

	subsRequest, err := json.Marshal(subscriptionRequest)
	slog.Info("Subscription Request", "Request", string(subsRequest))
	if err != nil {
		slog.Error("Error on parsing subscription request", "Error", err)
	}
	err = subscriber.feed.conn.WriteMessage(websocket.TextMessage, subsRequest)
	if err != nil {
		slog.Error("Error on sending subscription request", "Error", err)
	}

	return subscriber.getMarketDepth(ctx, currencyPair)
}

// get the market depth for a currency pair and populate the order book
func (subscriber *BinanceSubscriber) getMarketDepth(ctx context.Context, currPair string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(snapshotURL, currPair), nil)
	if err != nil {
		slog.Error("Error on Creating New GET Request", "curr pair", currPair, "Error", err)
		return err
	}

	client := &http.Client{}
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
	subscriber.feed.lastUpdateIds[currPair] = lastUpdateId

	firstUpdateId := <-subscriber.feed.updateIdChan
	slog.Info("last update id", "curr pair", currPair, "last update id", lastUpdateId, "first update id", firstUpdateId)
	if lastUpdateId > firstUpdateId {
		slog.Info("Condition Satisfied", "curr pair", currPair)
	} else {
		slog.Warn("Closing the Application. Re-get snapshot", "curr pair", currPair)
		return err
	}

	// orderbook.PopulateOrderBook(currPair, snapshot)
	return nil
}

// UnSubscribe to Currency Pair
func (subscriber *BinanceSubscriber) UnsubscribeToCurrPair(currencyPair string) {
	depthRequest := fmt.Sprintf(depthStr, strings.ToLower(currencyPair))
	unsubscriptionRequest := SubscriptionRequest{
		Method: unsubscribe,
		Params: []string{depthRequest},
		Id:     subscriber.uniqueIdGenerator.getUniqueReqId(),
	}

	slog.Info("Unsubscription currency pair", "curr pair", currencyPair)
	// orderbook.RemoveOrderBookForCurrency(currencyPair)

	unsubsRequest, err := json.Marshal(unsubscriptionRequest)
	slog.Info("UnSubscription Request", "request", string(unsubsRequest))
	if err != nil {
		slog.Error("Error on parsing unsubscription request", "error", err)
	}
	err = subscriber.feed.conn.WriteMessage(websocket.TextMessage, unsubsRequest)
	if err != nil {
		slog.Error("Error on sending unsubscription request", "error", err)
	}
	slog.Info("Unsubscribed currency pair", "curr pair", currencyPair)
}

// List of Subscribtions
func (subscriber *BinanceSubscriber) ListSubscriptions() []string {
	subscriber.feed.listSubscReqId = subscriber.uniqueIdGenerator.getUniqueReqId()
	subscriber.feed.listSubscriptions = make(chan []string)

	listSubscriptionRequest := ListSubscriptionRequest{
		Method: listSubscriptionsConst,
		Id:     subscriber.feed.listSubscReqId,
	}

	listsubsRequest, err := json.Marshal(listSubscriptionRequest)
	slog.Info("List Subscription Request", "request", string(listsubsRequest))
	if err != nil {
		slog.Error("Error on parsing list subscription request", "error", err)
	}
	err = subscriber.feed.conn.WriteMessage(websocket.TextMessage, listsubsRequest)
	if err != nil {
		slog.Error("Error on sending list subscription request", "error", err)
	}

	subscriptionsList := <-subscriber.feed.listSubscriptions
	return subscriptionsList
}
