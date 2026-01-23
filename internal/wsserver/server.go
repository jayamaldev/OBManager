package wsserver

import (
	"context"
	"log/slog"
	"net/http"
	"ob-manager/internal/subscriptions"

	"github.com/gorilla/websocket"
)

const (
	subscribe, unsubscribe = "SUB", "UNSUB"
)

type WSServer struct {
	srv       *http.Server
	processor *RequestProcessor
}

func NewWSServer(subs *subscriptions.Manager) *WSServer {
	proc := &RequestProcessor{
		subsManager: subs,
	}
	server := &http.Server{
		Addr:    ":8080",
		Handler: nil,
	}

	s := &WSServer{
		srv:       server,
		processor: proc,
	}

	go s.startServer()

	return s
}

func (s *WSServer) ShutDown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *WSServer) startServer() {
	http.HandleFunc("/ws", s.websocketHandler)
	slog.Info("Websocket Server started on :8080")

	err := s.srv.ListenAndServe()
	if err != nil {
		slog.Error("Error on websocket Server: ", "Error", err)
	}
}

func (s *WSServer) websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Error Upgrading Websocket: ", "Error", err)

		return
	}

	go s.processor.handleConnection(conn)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}
