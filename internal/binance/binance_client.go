package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	BinanceProcessor := NewBinanceProcessor()
	return &BinanceClient{
		processor: BinanceProcessor,
	}
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
		log.Fatal("Websocket connectivity issue", err)
		return err
	}
	fmt.Printf("Connected to websocket %s\n", u.String())

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
	fmt.Println("Websocket Client Closed")

	if err := g.Wait(); err != nil {
		log.Println("Error on Websocket Client")
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
			log.Println("Error on reading Websocket Message", err)
			return err
		}

		// market depth update
		err = json.Unmarshal(message, &eventUpdate)
		if err != nil {
			fmt.Println("Error Parsing Json", err)
			break
		}

		// subscription list response
		err = json.Unmarshal(message, &subscriptionsList)
		if err != nil {
			fmt.Println("Error Parsing Subscriptions List Json", err)
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
		log.Printf("Error on Creating New GET Request for curr paid %s\n", currPair)
		return err
	}

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)

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
	client.feed.lastUpdateIds[currPair] = lastUpdateId

	firstUpdateId := <-client.feed.updateIdChan
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

func (client *BinanceClient) CloseConnection() {
	err := client.feed.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("Error on writing close request to websocket")
	}
}
