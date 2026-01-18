package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ob-manager/internal/processors"
	"ob-manager/internal/queues/in"
	"ob-manager/internal/queues/out"
	"ob-manager/internal/subscriptions"
	"ob-manager/internal/upstream/binance"
	"ob-manager/internal/wsserver"
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

	// start upstream client and connect to market data provider
	initUpstreamClient(ctx, inQueue, procManager)

	// start downstream server
	server := startDownstreamServer(subManager)
	go server.StartServer()

	// start the push handler for the subscribed users
	subManager.StartPushHandler()

	gracefulShutdown(ctx, server)

	slog.Info("Exiting OrderBook Distributor Service")
}

func initUpstreamClient(ctx context.Context, queue *inqueues.InQManager, proc *processors.Manager) *binance.Client {
	slog.Info("Initializing Binance Client")

	requests := make(chan []byte)
	client := binance.NewClient(requests, queue, proc)

	go func() {
		client.ConnectToServer(ctx)
	}()

	go func() {
		err := client.SendRequests()
		if err != nil {
			slog.Error("Error in sending requests: ", err)
		}
	}()

	go func() {
		err := client.ProcessMessage()
		if err != nil {
			slog.Error("Error in processing message: ", err)
		}
	}()

	return client
}

// start websocket server.
func startDownstreamServer(sub *subscriptions.Manager) *wsserver.WSServer {
	processor := wsserver.NewProcessor(sub)
	server := wsserver.NewWSServer(processor)

	return server
}

// handle graceful shutdown.
func gracefulShutdown(ctx context.Context, server *wsserver.WSServer) {
	slog.Info("Graceful Shutdown is monitoring")

	<-ctx.Done()

	slog.Info("Shutdown Signal Received")

	var timeDuration = 30 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeDuration)

	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		slog.Error("Error in closing web socket server: ", "Error", err)
	}

	slog.Info("Server Exited Gracefully")
}
