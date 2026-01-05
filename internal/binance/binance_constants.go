package binance

const (
	subscribe, unsubscribe, listSubscriptionsConst = "SUBSCRIBE", "UNSUBSCRIBE", "LIST_SUBSCRIPTIONS"
	wssStream, binanceUrl, wsContextRoot           = "wss", "stream.binance.com:9443", "/ws"
	depthUpdateEvent                               = "depthUpdate"
)

var snapshotURL = "https://api.binance.com/api/v3/depth?symbol=%s&limit=50"

var depthStr = "%s@depth"
