package wsserver

import (
	"log/slog"
	"strings"

	"github.com/gorilla/websocket"
)

// abstraction of subscription user manager for the downstream server
type SubscriptionManager interface {
	AddNewUser(conn *websocket.Conn)
	RemoveUser(conn *websocket.Conn)
	SubUser(conn *websocket.Conn, currPair string, sub bool)
}

// abstraction of orderbook for the downstream server
type OBReader interface {
	GetOrderBook(curr string) []byte
}

type RequestProcessor struct {
	obReader OBReader
	subs     SubscriptionManager
}

func NewProcessor(obReader OBReader, subs SubscriptionManager) *RequestProcessor {
	return &RequestProcessor{
		obReader: obReader,
		subs:     subs,
	}
}

func (p *RequestProcessor) handleConnection(conn *websocket.Conn) {

	// remove user from the store when closing the connection
	defer func() {
		p.subs.RemoveUser(conn)
		err := conn.Close()
		if err != nil {
			slog.Error("Error on Closing the Connection", "error", err)
		}
	}()

	// separate user for each connection to manage subscriptions
	p.subs.AddNewUser(conn)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			slog.Error("Error Reading Message. ", "error", err)
			break
		}

		msgArgs := strings.Split(string(message), " ")

		slog.Info("Message Received: ", "message", string(message))

		switch string(msgArgs[0]) {
		case subscribe:
			currPair := msgArgs[1]
			p.handleSubscription(conn, currPair)
		case unsubscribe:
			currPair := msgArgs[1]
			p.handleUnsubscription(conn, currPair)
		default:
			slog.Info("Unknown command received")
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				slog.Error("Error Writing Message: ", "error", err)
			}
		}
	}
}

// handle user subscription request. add currency subscription to the user and send latest orderbook
func (p *RequestProcessor) handleSubscription(conn *websocket.Conn, currPair string) {
	slog.Info("Order Book Subscription Requested", "currency pair", currPair)
	err := conn.WriteMessage(websocket.TextMessage, p.obReader.GetOrderBook(currPair))
	if err != nil {
		slog.Error("Error Writing Message: ", "error", err)
	}

	p.subs.SubUser(conn, currPair, true)
}

// handle user unsubscription request. remove currency subscription from the user
func (p *RequestProcessor) handleUnsubscription(conn *websocket.Conn, currPair string) {
	slog.Info("Order Book Unsubscription Requested", "curr pair", currPair)
	p.subs.SubUser(conn, currPair, false)
}
