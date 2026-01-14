package binance

import "sync"

type IDGenerator struct {
	uniqueReqId int
	mutex       sync.Mutex
}

func NewIDGenerator() *IDGenerator {
	return &IDGenerator{}
}

// Function to get Unique request ID to send with Binance requests.
func (uniqueIDGen *IDGenerator) getUniqueReqId() int {
	uniqueIDGen.mutex.Lock()
	defer uniqueIDGen.mutex.Unlock()

	uniqueIDGen.uniqueReqId++

	return uniqueIDGen.uniqueReqId
}
