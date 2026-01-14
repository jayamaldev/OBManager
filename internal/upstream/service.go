package upstream

import (
	"context"
	"log/slog"

	"golang.org/x/sync/errgroup"
)

// Connector provides abstraction for the upstream client as the data source connector.
type Connector interface {
	ConnectToServer(g *errgroup.Group, bufferedMsgs chan []byte) error
	CloseConnection() error
	SendRequests() error
}

// Processor provides abstraction for the upstream client as the data source message processor.
type Processor interface {
	ProcessMessage(bufferedMsgs chan []byte) error
	UpdateEvents() error
	CloseProcessor()
}

// Subscriber provides abstraction for the upstream client as the Subs handler.
type Subscriber interface {
	SubscribeToCurrPair(ctx context.Context, currencyPair string) error
	UnsubscribeToCurrPair(currencyPair string)
	ListSubscriptions() []string
}

type Upstream struct {
	ws   Connector
	proc Processor
	subs Subscriber
}

func NewUpstream(ws Connector, proc Processor, subs Subscriber) *Upstream {
	return &Upstream{
		ws:   ws,
		proc: proc,
		subs: subs,
	}
}

// initUpstream initialize the upstream.
func (u *Upstream) initUpstream(ctx context.Context, g *errgroup.Group, currencyPair string) error {
	var bufferSize = 1000
	bufferedMsgs := make(chan []byte, bufferSize)

	err := u.ws.ConnectToServer(g, bufferedMsgs)
	if err != nil {
		slog.Error("Failed to connect to upstream server", "err", err)

		return err
	}

	// start a separate goroutine to process messages and add to a channel
	g.Go(func() error {
		return u.proc.ProcessMessage(bufferedMsgs)
	})

	// separate goroutine to send requests
	g.Go(func() error {
		return u.ws.SendRequests()
	})

	//separate goroutine to process events from the channel
	g.Go(func() error {
		return u.proc.UpdateEvents()
	})

	err = u.subs.SubscribeToCurrPair(ctx, currencyPair)
	if err != nil {
		slog.Error("Failed to subscribe to currency pair", "err", err)

		return err
	}

	return nil
}

func (u *Upstream) closeUpstream() error {
	u.proc.CloseProcessor()

	return u.ws.CloseConnection()
}
