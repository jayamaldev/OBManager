package subscriptions

import (
	"encoding/json"
	"log/slog"
	"slices"

	"ob-manager/internal/dtos"

	"github.com/gorilla/websocket"
)

type OutQGetter interface {
	OutQ() <-chan *dtos.EventUpdate
}

type OBGetter interface {
	GetOrderBook(curr string) ([]byte, int)
}

type User struct {
	conn         *websocket.Conn
	lastUpdateId int
}

func NewUser(conn *websocket.Conn) *User {
	return &User{
		conn: conn,
	}
}

type Manager struct {
	OutQGetter
	OBGetter

	subs map[string][]*User
}

func NewManager(getter OutQGetter, obGetter OBGetter) *Manager {
	return &Manager{
		OutQGetter: getter,
		OBGetter:   obGetter,
		subs:       make(map[string][]*User),
	}
}

// AddSubscription adds a subscription for the user to a currency pair.
func (m *Manager) AddSubscription(currency string, conn *websocket.Conn) {
	if m.subs[currency] == nil {
		m.subs[currency] = make([]*User, 0)
	}

	user := NewUser(conn)

	m.subs[currency] = append(m.subs[currency], user)

	slog.Info("User Subscribed", "Currency", currency)
}

// RemoveSubscription removes a subscription for the user to a currency pair.
func (m *Manager) RemoveSubscription(currency string, conn *websocket.Conn) {
	index := -1

	for i, u := range m.subs[currency] {
		if u.conn == conn {
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

	for _, u := range m.subs[event.Symbol] {
		if u.lastUpdateId == 0 {
			// send order book
			ob, lastUpdateId := m.GetOrderBook(event.Symbol)
			m.sendWSMessage(u.conn, ob)
			u.lastUpdateId = lastUpdateId //set order book id
		}

		if event.FinalUpdateId > u.lastUpdateId {
			u.lastUpdateId = event.FinalUpdateId
			m.sendWSMessage(u.conn, message)
		}
	}
}

func (m *Manager) sendWSMessage(conn *websocket.Conn, message []byte) {
	err := conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		slog.Error("Error on Writing to Websocket", "Error", err)
	}
}
