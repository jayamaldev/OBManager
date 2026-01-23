package dtos

// SubscriptionRequest to the Binance to Subscribe/ Unsubscribe for a Currency Pair.
type SubscriptionRequest struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	Id     int32    `json:"id"`
}

// ListSubscriptionRequest Request to get the Subscribed Currency Pairs list from Binance.
type ListSubscriptionRequest struct {
	Method string `json:"method"`
	Id     int32  `json:"id"`
}

type SubscriptionsList struct {
	Result []string `json:"result"`
	Id     int      `json:"id"`
}
