package main

import (
	"context"
	"log/slog"
	"ob-manager/internal/processors"
	"ob-manager/internal/subscriptions"
	"ob-manager/internal/upstream/binance"
	"ob-manager/internal/wsserver"
	"os"
	"os/signal"
	"syscall"
	"time"

	inqueues "ob-manager/internal/queues/in"
	outqueues "ob-manager/internal/queues/out"
)

func main() {
	slog.Info("Starting Binance Distributor Service")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// in queues manager
	inQueue := inqueues.NewQManager()

	// out queue manager
	outQueue := outqueues.NewQueue()

	// order book processes Manager
	procManager := processors.NewManager(inQueue, outQueue)

	// initialize downstream subscribers store
	subManager := subscriptions.NewManager(outQueue, procManager)

	// start upstream client and connect to the market data provider
	initUpstreamClient(ctx, inQueue, procManager)

	// start a downstream server
	server := startDownstreamServer(subManager)

	gracefulShutdown(ctx, server)

	slog.Info("Exiting OrderBook Distributor Service")
}

func initUpstreamClient(ctx context.Context, queue *inqueues.InQManager, proc *processors.Manager) *binance.Client {
	slog.Info("Initializing Binance Client")

	requests := make(chan []byte)
	client := binance.NewClient(requests, queue, proc)
	client.StartClient(ctx)

	return client
}

// start websocket server.
func startDownstreamServer(sub *subscriptions.Manager) *wsserver.WSServer {
	return wsserver.NewWSServer(sub)
}

// handle a graceful shutdown.
func gracefulShutdown(ctx context.Context, server *wsserver.WSServer) {
	slog.Info("Graceful Shutdown is monitoring")

	<-ctx.Done()

	slog.Info("Shutdown Signal Received")

	timeDuration := 30 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeDuration)

	defer cancel()

	err := server.ShutDown(ctx)
	if err != nil {
		slog.Error("Error in closing web socket server: ", "Error", err)
	}

	slog.Info("Server Exited Gracefully")
}
