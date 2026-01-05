package main

import (
	"fmt"
	"ob-manager/internal/binance"
)

func main() {
	fmt.Println("Hello")

	binanceClient := binance.NewBinanceClient()
	binanceClient.SetCurrencyPair("BTCUSDT")
	binanceClient.ConnectToWebSocket()
}
