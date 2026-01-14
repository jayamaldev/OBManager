package wsserver

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
)

const (
	subscribe, unsubscribe = "SUB", "UNSUB"
)

type Processor interface {
	handleConnection(conn *websocket.Conn)
}

type WSServer struct {
	*http.Server
	Processor
}

func NewWSServer(proc Processor) *WSServer {
	server := &http.Server{
		Addr:    ":8080",
		Handler: nil,
	}

	return &WSServer{
		Server:    server,
		Processor: proc,
	}
}

func (s *WSServer) StartServer() error {
	http.HandleFunc("/ws", s.websocketHandler)
	slog.Info("Websocket Server started on :8080")

	err := s.ListenAndServe()
	if err != nil {
		slog.Error("Error on websocket Server: ", "Error", err)

		return err
	}

	return nil
}

func (s *WSServer) ShutDown(ctx context.Context) error {
	return s.Shutdown(ctx)
}

func (s *WSServer) websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Error Upgrading Websocket: ", "Error", err)

		return
	}

	go s.handleConnection(conn)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}
