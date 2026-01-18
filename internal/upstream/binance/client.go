package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"ob-manager/internal/dtos"
	"strings"

	"github.com/gorilla/websocket"
)

type OBUpdater interface {
	UpdateBids(currency string, bids map[float64]float64)
	UpdateAsks(currency string, asks map[float64]float64)
}

type SnapshotGetter interface {
	GetSnapshot(ctx context.Context, currPair string) error
}

type QueueAdder interface {
	AddToQueue(update *dtos.EventUpdate)
}

type Client struct {
	QueueAdder
	SnapshotGetter
	OBUpdater

	conn         *websocket.Conn
	requests     chan []byte
	idGen        *IDGenerator
	bufferedMsgs chan []byte
}

func NewClient(requests chan []byte, q QueueAdder, updater OBUpdater) *Client {
	idGen := NewIDGenerator()
	getter := NewRestClient(updater)
	bufferSize := 20000

	return &Client{
		idGen:          idGen,
		requests:       requests,
		bufferedMsgs:   make(chan []byte, bufferSize),
		QueueAdder:     q,
		SnapshotGetter: getter,
	}
}

// ConnectToServer todo handle disconnections.
func (c *Client) ConnectToServer() error {
	u := url.URL{
		Scheme: wssStream,
		Host:   binanceUrl,
		Path:   wsContextRoot,
	}

	slog.Info("connecting to websocket", "url", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)

	if err != nil {
		slog.Error("Websocket connectivity issue", "Error", err)

		return err
	}

	slog.Info("Connected to websocket", "url", u.String())

	c.conn = conn

	go func() {
		err := c.readWSMessages()
		if err != nil {
			slog.Error("Websocket read error", "Error", err) // todo handle this error.
		}
	}()

	return nil
}

func (c *Client) SendRequests() error {
	for {
		request := <-c.requests
		slog.Info("Sending Web Socket Request", "Request", request)
		err := c.conn.WriteMessage(websocket.TextMessage, request)

		if err != nil {
			slog.Error("Error on sending subscription request", "Error", err)

			return err
		}
	}
}

func (c *Client) CloseConnection() error {
	close(c.requests)

	err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	if err != nil {
		slog.Error("Error on writing close request to websocket", "Error", err)

		return err
	}

	return nil
}

func (c *Client) SubscribeToCurrPair(currencyPair string) error {
	depthRequest := fmt.Sprintf(depthStr, strings.ToLower(currencyPair))
	subscriptionRequest := dtos.SubscriptionRequest{
		Method: subscribe,
		Params: []string{depthRequest},
		Id:     c.idGen.getUniqueReqId(),
	}

	slog.Info("Subscription currency pair", "curr pair", currencyPair)

	subsRequest, err := json.Marshal(subscriptionRequest)
	slog.Info("Subscription", "Request", string(subsRequest))

	if err != nil {
		slog.Error("Error on parsing subscription request", "Error", err)
	}

	c.requests <- subsRequest

	return nil
}

func (c *Client) ProcessMessage() error {
	slog.Info("started binance message processors")

	var rawMap map[string]interface{}

	for {
		var (
			eventUpdate       dtos.EventUpdate
			subscriptionsList dtos.SubscriptionsList
		)

		slog.Debug("Waiting for messages from binance")

		message := <-c.bufferedMsgs

		err := json.Unmarshal(message, &rawMap)
		if err != nil {
			slog.Error("Error on unmarshalling message", "Error", err)

			break
		}

		_, ok := rawMap["id"].(string)

		if ok {
			// subscription list response
			err = json.Unmarshal(message, &subscriptionsList)
			if err != nil {
				slog.Error("Error Parsing Subscriptions List Json", "Error", err)

				break
			}

			c.processSubscriptionList(subscriptionsList, message)
		} else {
			// market depth update
			err := json.Unmarshal(message, &eventUpdate)
			if err != nil {
				slog.Error("Error Parsing Json", "Error", err)

				break
			}

			c.processMarketDepthUpdate(eventUpdate)
		}
	}

	return nil
}

func (c *Client) readWSMessages() error {
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			slog.Error("Error on reading Websocket Message", "Error", err)

			return err
		}

		slog.Info("WS message received.", "Message", message)

		if len(message) > 0 {
			c.bufferedMsgs <- message
		}
	}
}

func (c *Client) processMarketDepthUpdate(eventUpdate dtos.EventUpdate) {
	c.AddToQueue(&eventUpdate)
	slog.Debug("adding event to the channel")
}

func (c *Client) processSubscriptionList(subscriptionsList dtos.SubscriptionsList, message []byte) {
	slog.Info("admin message received: ", "message", string(message), "Id", subscriptionsList.Id)
}
