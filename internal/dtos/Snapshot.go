package dtos

type Snapshot struct {
	LastUpdateId int        `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

// FEEDBACK: Snapshot.go Go files should have simple cases without underscores or mixedCaps. Consider renaming to snapshot.go
