package upstream

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// data source connector abstraction for the upstream client
type Connector interface {
	ConnectToServer(g *errgroup.Group, bufferedMsgs chan []byte) error
	CloseConnection() error
	SendRequests() error
}

// data source message processor abstraction for the upstream client
type Processor interface {
	ProcessMessage(bufferedMsgs chan []byte) error
	UpdateEvents() error
	CloseProcessor()
}

// Subs handler abstraction for the upstream client
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

// initialize the upstream
func (u *Upstream) initUpstream(ctx context.Context, g *errgroup.Group, currencyPair string) {
	bufferedMsgs := make(chan []byte, 1000)

	u.ws.ConnectToServer(g, bufferedMsgs)

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

	u.subs.SubscribeToCurrPair(ctx, currencyPair)
}

func (u *Upstream) closeUpstream() error {
	u.proc.CloseProcessor()
	return u.ws.CloseConnection()
}
