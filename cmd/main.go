package main

import (
	"context"
	"log/slog"
	"ob-manager/internal/downstream"
	"ob-manager/internal/downstream/wsserver"
	"ob-manager/internal/orderbook"
	"ob-manager/internal/subscribers"
	"ob-manager/internal/upstream"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	subHandler := initSusbscriptionHandler()

	// start upstream client and connect to market data provider
	client := startUpstreamClient(cg, storeHandler, subHandler)

	// start downstream server
	server := startDownstreamServer(cg, storeHandler, subHandler)

	g.Go(func() error {
		return gracefulShutdown(cg, client, server)
	})

	if err := g.Wait(); err != nil {
		slog.Error("Error on OrderBook Distributor Service", "Error", err)
	}

	slog.Info("Exiting OrderBook Distributor Service")
}

// initialize order book store
func initOrderBookStore() *orderbook.StoreHandler {
	obstore := orderbook.NewStore()
	return orderbook.NewStoreHandler(obstore)
}

// initialize downstream subscribers store
func initSusbscriptionHandler() *subscribers.Handler {
	subsStore := subscribers.NewUserStore()
	return subscribers.NewHandler(subsStore)
}

// start websocket server with error group
func startUpstreamClient(cg *ContextGroup, ob *orderbook.StoreHandler, subs *subscribers.Handler) *upstream.Client {
	client := upstream.NewClient(ob, subs)
	client.SetMainCurrencyPair("BTCUSDT")
	client.InitClient(cg.g, cg.ctx)
	return client
}

// start websocket server with error group
func startDownstreamServer(cg *ContextGroup, ob *orderbook.StoreHandler, subs *subscribers.Handler) *downstream.Handler {
	processor := wsserver.NewProcessor(ob, subs)
	server := wsserver.NewWSServer(processor)
	dsHandler := downstream.NewHandler(server)

	cg.g.Go(func() error {
		if err := dsHandler.StartServer(); err != nil {
			slog.Error("error on starting Downstream server", "Error", err)
		}
		slog.Info("Downstream Server Initialized")
		return nil
	})
	return dsHandler
}

// handle graceful shutdown
func gracefulShutdown(cg *ContextGroup, client *upstream.Client, dsHandler *downstream.Handler) error {
	slog.Info("Gradeful Shutdown is monitoring")

	<-cg.ctx.Done()

	slog.Info("Shutdown Signal Received")

	ctx, cancel := context.WithTimeout(cg.ctx, 30*time.Second)
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
