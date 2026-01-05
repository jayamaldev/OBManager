package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"ob-manager/internal/binance/dtos"
	"strings"

	"github.com/gorilla/websocket"
)

type BinanceSubscriber struct {
	feed              *BinanceFeed
	uniqueIdGenerator *IDGenerator
}

func NewBinanceSubscriber() *BinanceSubscriber {
	idGen := NewIDGenerator()
	return &BinanceSubscriber{
		uniqueIdGenerator: idGen,
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

	fmt.Println("Subscription currency pair: ", currencyPair)
	// orderbook.InitOrderBookForCurrency(currencyPair)
	subscriber.feed.firstEntryMap[currencyPair] = true

	subsRequest, err := json.Marshal(subscriptionRequest)
	fmt.Println("Subscription Request: ", string(subsRequest))
	if err != nil {
		fmt.Println("Error on parsing subscription request", err)
	}
	err = subscriber.feed.conn.WriteMessage(websocket.TextMessage, subsRequest)
	if err != nil {
		fmt.Println("error on sending subscription request")
	}

	return subscriber.getMarketDepth(ctx, currencyPair)
}

// get the market depth for a currency pair and populate the order book
func (subscriber *BinanceSubscriber) getMarketDepth(ctx context.Context, currPair string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(snapshotURL, currPair), nil)
	if err != nil {
		log.Printf("Error on Creating New GET Request for curr paid %s\n", currPair)
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Printf(fmt.Sprintf("Error on Closing Response for curr pair %s", currPair), err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		log.Fatal(fmt.Sprintf("Error on Getting Snapshot for curr pair %s", currPair), err)
		return err
	}

	var snapshot *dtos.Snapshot
	err = json.NewDecoder(resp.Body).Decode(&snapshot)
	if err != nil {
		log.Fatal(fmt.Sprintf("Cound not parse Response Json for curr pair %s", currPair), err)
		return err
	}

	lastUpdateId := snapshot.LastUpdateId
	subscriber.feed.lastUpdateIds[currPair] = lastUpdateId

	firstUpdateId := <-subscriber.feed.updateIdChan
	fmt.Printf("last update id for currency pair %s : %d first update id %d \n", currPair, lastUpdateId, firstUpdateId)
	if lastUpdateId > firstUpdateId {
		fmt.Printf("Condition Satisfied for curr pair %s !! \n", currPair)
	} else {
		fmt.Printf("Closing the Application. Re-get snapshot for currency pair %s\n", currPair)
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

	fmt.Println("Unsubscription currency pair: ", currencyPair)
	// orderbook.RemoveOrderBookForCurrency(currencyPair)

	unsubsRequest, err := json.Marshal(unsubscriptionRequest)
	fmt.Println("UnSubscription Request: ", string(unsubsRequest))
	if err != nil {
		fmt.Println("Error on parsing unsubscription request", err)
	}
	err = subscriber.feed.conn.WriteMessage(websocket.TextMessage, unsubsRequest)
	if err != nil {
		fmt.Println("error on sending unsubscription request")
	}
	fmt.Println("Unsubscribed for ", currencyPair)
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
	fmt.Println("List Subscription Request: ", string(listsubsRequest))
	if err != nil {
		fmt.Println("Error on parsing list subscription request", err)
	}
	err = subscriber.feed.conn.WriteMessage(websocket.TextMessage, listsubsRequest)
	if err != nil {
		fmt.Println("error on sending unsubscription request")
	}

	subscriptionsList := <-subscriber.feed.listSubscriptions
	return subscriptionsList
}
