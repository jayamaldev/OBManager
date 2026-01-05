package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"ob-manager/internal/binance/dtos"

	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

type BinanceFeed struct {
	conn              *websocket.Conn
	firstEntryMap     map[string]bool
	bufferedEvents    chan dtos.EventUpdate
	updateIdChan      chan int
	lastUpdateIds     map[string]int
	listSubscReqId    int
	listSubscriptions chan []string
}

type BinanceClient struct {
	feed             *BinanceFeed
	processor        *BinanceProcessor
	subscriber       *BinanceSubscriber
	mainCurrencyPair string
}

func NewBinanceClient() *BinanceClient {
	firstEntryMap := make(map[string]bool)
	lastUpdateIds := make(map[string]int)
	bufferedEvents := make(chan dtos.EventUpdate, 1000)
	updateIdChan := make(chan int)
	listSubscriptions := make(chan []string)

	feed := &BinanceFeed{
		firstEntryMap:     firstEntryMap,
		lastUpdateIds:     lastUpdateIds,
		bufferedEvents:    bufferedEvents,
		updateIdChan:      updateIdChan,
		listSubscriptions: listSubscriptions,
	}
	binanceProcessor := NewBinanceProcessor(feed)
	binanceSubscriber := NewBinanceSubscriber(feed)
	return &BinanceClient{
		processor:  binanceProcessor,
		feed:       feed,
		subscriber: binanceSubscriber,
	}
}

func (client *BinanceClient) SetCurrencyPair(currPair string) {
	client.mainCurrencyPair = currPair
}

func (client *BinanceClient) ConnectToWebSocket() error {
	u := url.URL{
		Scheme: wssStream,
		Host:   binanceUrl,
		Path:   wsContextRoot,
	}

	g, ctx := errgroup.WithContext(context.Background())
	slog.Info("connecting to websocket", "url", u.String())

	client.feed.firstEntryMap = make(map[string]bool)

	var err error
	client.feed.conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		slog.Error("Websocket connectivity issue", "Error", err)
		return err
	}
	slog.Info("Connected to websocket", "url", u.String())

	done := make(chan struct{})
	client.feed.updateIdChan = make(chan int)
	client.feed.bufferedEvents = make(chan dtos.EventUpdate, 1000)

	defer close(client.feed.bufferedEvents)
	defer close(client.feed.updateIdChan)

	g.Go(func() error {
		defer close(done)
		return client.readAndProcessWSMessages()
	})

	err = client.subscriber.SubscribeToCurrPair(ctx, client.mainCurrencyPair)
	if err != nil {
		return err
	}

	go client.processor.updateEvents()
	<-done
	slog.Info("Websocket Client Closed")

	if err := g.Wait(); err != nil {
		slog.Error("Error on Websocket Client", "Error", err)
		return err
	}

	return nil
}

func (client *BinanceClient) readAndProcessWSMessages() error {
	for {
		var eventUpdate dtos.EventUpdate
		var subscriptionsList dtos.SubscriptionsList

		_, message, err := client.feed.conn.ReadMessage()
		if err != nil {
			slog.Error("Error on reading Websocket Message", "Error", err)
			return err
		}

		// market depth update
		err = json.Unmarshal(message, &eventUpdate)
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
		client.processor.processMarketDepthUpdate(eventUpdate, message)

		// process list of subscription responses
		client.processor.processSubscriptionList(subscriptionsList, message)
	}
	return nil
}

// get the market depth for a currency pair and populate the order book
func (client *BinanceClient) getMarketDepth(ctx context.Context, currPair string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(snapshotURL, currPair), nil)
	if err != nil {
		slog.Error("Error on Creating New GET Request for curr pair", "currency", currPair, "Error", err)
		return err
	}

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			slog.Error("Error on Closing Response for curr pair", "currency", currPair, "Error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		slog.Error("Error on Getting Snapshot for curr pair", "currency", currPair, "Error", err)
		return err
	}

	var snapshot *dtos.Snapshot
	err = json.NewDecoder(resp.Body).Decode(&snapshot)
	if err != nil {
		slog.Error("Cound not parse Response Json for curr pair", "currency", currPair, "Error", err)
		return err
	}

	lastUpdateId := snapshot.LastUpdateId
	client.feed.lastUpdateIds[currPair] = lastUpdateId

	firstUpdateId := <-client.feed.updateIdChan
	slog.Info("last update id for ", "currency", currPair, "last update id", lastUpdateId, "first update id", firstUpdateId)
	if lastUpdateId > firstUpdateId {
		slog.Info("Condition Satisfied", "currency pair", currPair)
	} else {
		slog.Info("Closing the Application. Re-get snapshot", "currency pair", currPair)
		return err
	}

	// orderbook.PopulateOrderBook(currPair, snapshot)
	return nil
}

func (client *BinanceClient) CloseConnection() {
	err := client.feed.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		slog.Error("CError on writing close request to websocket", "Error", err)
	}
}
