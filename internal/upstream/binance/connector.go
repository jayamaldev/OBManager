package binance

import (
	"log/slog"
	"net/url"

	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

type WSClient struct {
	conn     *websocket.Conn
	requests chan []byte
}

func NewWSClient(requests chan []byte) *WSClient {
	return &WSClient{
		requests: requests,
	}
}

func (ws *WSClient) ConnectToServer(g *errgroup.Group, bufferedMsgs chan []byte) error {
	u := url.URL{
		Scheme: wssStream,
		Host:   binanceUrl,
		Path:   wsContextRoot,
	}

	slog.Info("connecting to websocket", "url", u.String())

	var err error
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)

	if err != nil {
		slog.Error("Websocket connectivity issue", "Error", err)

		return err
	}

	slog.Info("Connected to websocket", "url", u.String())

	ws.conn = conn

	g.Go(func() error {
		return ws.readWSMessages(bufferedMsgs)
	})

	return nil
}

func (ws *WSClient) SendRequests() error {
	for {
		request := <-ws.requests
		slog.Info("Sending Web Socket Requst", "Request", request)
		err := ws.conn.WriteMessage(websocket.TextMessage, request)

		if err != nil {
			slog.Error("Error on sending subscription request", "Error", err)

			return err
		}
	}
}

func (ws *WSClient) CloseConnection() error {
	close(ws.requests)
	err := ws.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	if err != nil {
		slog.Error("Error on writing close request to websocket", "Error", err)

		return err
	}

	return nil
}

func (ws *WSClient) readWSMessages(bufferedMsgs chan []byte) error {
	for {
		_, message, err := ws.conn.ReadMessage()
		if err != nil {
			slog.Error("Error on reading Websocket Message", "Error", err)

			return err
		}

		slog.Info("WS message received.", "Message", message)

		if len(message) > 0 {
			bufferedMsgs <- message
		}
	}
}
