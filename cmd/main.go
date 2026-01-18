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

	// in queue manager
	inQueue := inqueues.NewQManager()

	// out queue manager
	outQueue := outqueues.NewQueue()

	// initialize downstream subscribers store
	subManager := subscriptions.NewManager(outQueue)

	// process Managers
	procManager := processors.NewManager(inQueue, outQueue)

	// start upstream client and connect to market data provider
	client := initUpstreamClient(inQueue, procManager)

	// start downstream server
	server := startDownstreamServer(procManager, subManager)
	go server.StartServer()

	// subscribe to default currency list
	subscribeToCurrencies(ctx, client, procManager)

	// start the push handler for the subscribed users
	subManager.StartPushHandler()

	gracefulShutdown(ctx, server)

	slog.Info("Exiting OrderBook Distributor Service")
}

func initUpstreamClient(queue *inqueues.InQManager, proc *processors.Manager) *binance.Client {
	slog.Info("Initializing Binance Client")

	requests := make(chan []byte)
	client := binance.NewClient(requests, queue, proc)

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

	err := client.ConnectToServer()
	if err != nil {
		slog.Error("Error in connecting to server: ", err)
	}

	return client
}

// start websocket server.
func startDownstreamServer(ob *processors.Manager, sub *subscriptions.Manager) *wsserver.WSServer {
	processor := wsserver.NewProcessor(ob, sub)
	server := wsserver.NewWSServer(processor)

	return server
}

// subscribeToCurrencies todo get currencies list from configurations.
func subscribeToCurrencies(ctx context.Context, client *binance.Client, procManager *processors.Manager) {
	currencies := []string{"BTCUSDT", "ETHUSDT"}

	for _, currency := range currencies {
		slog.Info("Subscribing", "currency", currency)

		go func(curr string) {
			err := client.SubscribeToCurrPair(curr)
			if err != nil {
				slog.Error("Error in subscribing", "Currency", currency, err)
			}
		}(currency)

		go func(ctx context.Context, curr string) {
			err := client.GetSnapshot(ctx, curr)
			if err != nil {
				slog.Error("Error in getting snapshot", "Currency", currency, err)
			}
		}(ctx, currency)

		go func(curr string) {
			procManager.StartProcessor(curr)
		}(currency)
	}
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
