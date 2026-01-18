package subscriptions

import (
	"encoding/json"
	"log/slog"
	"ob-manager/internal/dtos"
	"slices"

	"github.com/gorilla/websocket"
)

type OutQGetter interface {
	OutQ() <-chan *dtos.EventUpdate
}

type Manager struct {
	OutQGetter

	subs map[string][]*websocket.Conn
}

func NewManager(getter OutQGetter) *Manager {
	return &Manager{
		OutQGetter: getter,
		subs:       make(map[string][]*websocket.Conn),
	}
}

// AddSubscription adds a subscription for the user to a currency pair.
func (m *Manager) AddSubscription(currency string, conn *websocket.Conn) {
	if m.subs[currency] == nil {
		m.subs[currency] = make([]*websocket.Conn, 0)
	}

	m.subs[currency] = append(m.subs[currency], conn)

	slog.Info("User Subscribed", "Currency", currency)
}

// RemoveSubscription removes a subscription for the user to a currency pair.
func (m *Manager) RemoveSubscription(currency string, conn *websocket.Conn) {
	index := -1

	for i, c := range m.subs[currency] {
		if c == conn {
			index = i

			break
		}
	}

	if index != -1 {
		m.subs[currency] = slices.Delete(m.subs[currency], index, index+1)

		slog.Info("Subscription Removed", "Currency", currency)
	}
}

// RemoveUser removes all the subscriptions from a user.
func (m *Manager) RemoveUser(conn *websocket.Conn) {
	for curr := range m.subs {
		m.RemoveSubscription(curr, conn)
	}
}

// StartPushHandler creates a go routine that handles push messages to the subscribers.
func (m *Manager) StartPushHandler() {
	go func() {
		slog.Info("Starting Push Handler")

		for {
			event := <-m.OutQ()
			m.handlePushEvent(event)
		}
	}()
}

func (m *Manager) handlePushEvent(event *dtos.EventUpdate) {
	message, err := json.Marshal(event)

	if err != nil {
		slog.Error("error on parsing push event to json", "Error", err)
	}

	for _, conn := range m.subs[event.Symbol] {
		err := conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			slog.Error("Error on Writing to Websocket", "Error", err)
		}
	}
}
