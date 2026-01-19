package dtos

type SubscriptionsList struct {
	Result []string `json:"result"`
	Id     int      `json:"id"`
}

// FEEDBACK:  Why not subscriptions.go ?
