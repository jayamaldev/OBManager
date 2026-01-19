package wsserver

import (
	"log/slog"
	"strings"

	"github.com/gorilla/websocket"
)

// SubscriptionManager provides abstraction of subscription user manager for the downstream server.
type SubscriptionManager interface {
	AddSubscription(currency string, conn *websocket.Conn)
	RemoveSubscription(currency string, conn *websocket.Conn)
	RemoveUser(conn *websocket.Conn)
}

type RequestProcessor struct {
	SubscriptionManager // FEEDBACK: Why embedding the interface here ? This will expose the methods of SubscriptionManager on RequestProcessor.
}

func NewProcessor(subs SubscriptionManager) *RequestProcessor {
	return &RequestProcessor{
		SubscriptionManager: subs,
	}
}

func (p *RequestProcessor) handleConnection(conn *websocket.Conn) {
	// remove user from the store when closing the connection
	defer func() {
		p.RemoveUser(conn)
		err := conn.Close()
		if err != nil {
			slog.Error("Error on Closing the Connection", "error", err)
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			slog.Error("Error Reading Message. ", "error", err)

			break
		}

		msgArgs := strings.Split(string(message), " ")

		slog.Info("Message Received: ", "message", string(message))

		switch msgArgs[0] {
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

// handle user subscription request. add currency subscription to the user and send latest order book.
func (p *RequestProcessor) handleSubscription(conn *websocket.Conn, currPair string) {
	slog.Info("Order Book Subscription Requested", "currency pair", currPair)
	p.AddSubscription(currPair, conn)
}

// handle user unsubscription request. remove currency subscription from the user.
func (p *RequestProcessor) handleUnsubscription(conn *websocket.Conn, currPair string) {
	slog.Info("Order Book Unsubscription Requested", "curr pair", currPair)
	p.RemoveSubscription(currPair, conn)
}
