package dtos

type EventUpdate struct {
	EventType     string     `json:"e"`
	EventTime     int        `json:"E"`
	Symbol        string     `json:"s"`
	FirstUpdateId int        `json:"U"`
	FinalUpdateId int        `json:"u"`
	Bids          [][]string `json:"b"`
	Asks          [][]string `json:"a"`
}
