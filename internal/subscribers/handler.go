package subscribers

import (
	"log/slog"

	"github.com/gorilla/websocket"
)

// Store provides abstraction of the subscribed user store for the application.
type Store interface {
	AddUser(conn *websocket.Conn, user *User)
	RemoveUser(conn *websocket.Conn)
	SubUser(conn *websocket.Conn, currPair string, sub bool)
	GetSubscribedUsedList(currency string) []*User
}

type User struct {
	currPairs []string
	conn      *websocket.Conn
}

type Handler struct {
	Store
}

func NewHandler(store Store) *Handler {
	return &Handler{
		Store: store,
	}
}

func (h *Handler) AddNewUser(conn *websocket.Conn) {
	user := &User{
		conn: conn,
	}
	h.AddUser(conn, user)
}

func (h *Handler) PushEventToUsers(message []byte, currency string) {
	subUsersList := h.GetSubscribedUsedList(currency)
	for _, user := range subUsersList {
		err := user.conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			slog.Error("Error on Writing to Websocket", "Error", err)
		}
	}
}
