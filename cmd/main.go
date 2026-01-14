package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ob-manager/internal/downstream"
	"ob-manager/internal/downstream/wsserver"
	"ob-manager/internal/orderbook"
	"ob-manager/internal/subscribers"
	"ob-manager/internal/upstream"

	"golang.org/x/sync/errgroup"
)

type ContextGroup struct {
	g   *errgroup.Group
	ctx context.Context
}

func NewContextGroup(ctx context.Context, g *errgroup.Group) *ContextGroup {
	return &ContextGroup{
		g:   g,
		ctx: ctx,
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	g, ctx := errgroup.WithContext(ctx)
	cg := NewContextGroup(ctx, g)

	// initialize order book store
	storeHandler := initOrderBookStore()

	// initialize downstream subscribers store
	subHandler := initSubscriptionHandler()

	// start upstream client and connect to market data provider
	client := initUpstreamClient(storeHandler, subHandler)

	g.Go(func() error {
		return client.InitClient(cg.g, cg.ctx)
	})

	// start downstream server
	server := startDownstreamServer(storeHandler, subHandler)

	g.Go(func() error {
		return server.StartServer()
	})

	g.Go(func() error {
		return gracefulShutdown(cg, client, server)
	})

	if err := g.Wait(); err != nil {
		slog.Error("Error on OrderBook Distributor Service", "Error", err)
	}

	slog.Info("Exiting OrderBook Distributor Service")
}

// initialize order book store.
func initOrderBookStore() *orderbook.StoreHandler {
	obStore := orderbook.NewStore()

	return orderbook.NewStoreHandler(obStore)
}

// initialize downstream subscribers store.
func initSubscriptionHandler() *subscribers.Handler {
	subsStore := subscribers.NewUserStore()

	return subscribers.NewHandler(subsStore)
}

// start websocket server with error group.
func initUpstreamClient(ob *orderbook.StoreHandler, subs *subscribers.Handler) *upstream.Client {
	client := upstream.NewClient(ob, subs)
	client.SetMainCurrencyPair("BTCUSDT")

	return client
}

// start websocket server with error group.
func startDownstreamServer(ob *orderbook.StoreHandler, sub *subscribers.Handler) *downstream.Handler {
	processor := wsserver.NewProcessor(ob, sub)
	server := wsserver.NewWSServer(processor)
	dsHandler := downstream.NewHandler(server)

	return dsHandler
}

// handle graceful shutdown.
func gracefulShutdown(cg *ContextGroup, client *upstream.Client, dsHandler *downstream.Handler) error {
	slog.Info("Graceful Shutdown is monitoring")

	<-cg.ctx.Done()

	slog.Info("Shutdown Signal Received")

	var timeDuration = 30 * time.Second
	ctx, cancel := context.WithTimeout(cg.ctx, timeDuration)

	defer cancel()

	err := client.CloseClient()
	if err != nil {
		slog.Error("Forced Shutdown: ", "Error", err)

		return err
	}

	slog.Info("Websocket Client Closed")

	err = dsHandler.ShutDown(ctx)
	if err != nil {
		slog.Error("Forced Shutdown: ", "Error", err)
	}

	slog.Info("Websocket Server Closed")

	slog.Info("Server Exited Gracefully")

	return nil
}
