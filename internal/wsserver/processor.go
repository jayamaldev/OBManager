package wsserver

import (
	"log/slog"
	"ob-manager/internal/subscriptions"
	"strings"

	"github.com/gorilla/websocket"
)

type RequestProcessor struct {
	subsManager *subscriptions.Manager
}

func (p *RequestProcessor) handleConnection(conn *websocket.Conn) {
	// remove the user from the store when closing the connection
	defer func() {
		p.subsManager.RemoveUser(conn)
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

// handle user subscription request. add the currency subscription to the user and send the latest order book.
func (p *RequestProcessor) handleSubscription(conn *websocket.Conn, currPair string) {
	slog.Info("Order Book Subscription Requested", "currency pair", currPair)
	p.subsManager.AddSubscription(currPair, conn)
}

// handle user unsubscription request. remove currency subscription from the user.
func (p *RequestProcessor) handleUnsubscription(conn *websocket.Conn, currPair string) {
	slog.Info("Order Book Unsubscription Requested", "curr pair", currPair)
	p.subsManager.RemoveSubscription(currPair, conn)
}
