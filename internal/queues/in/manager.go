package inqueues

import (
	"ob-manager/internal/dtos"
	"sync"
)

const (
	queueSize = 10000
)

type InQManager struct {
	mu     sync.Mutex
	queues map[string]chan *dtos.EventUpdate
}

func NewQManager() *InQManager {
	return &InQManager{
		queues: make(map[string]chan *dtos.EventUpdate),
	}
}

func (m *InQManager) AddToQueue(eventUpdate *dtos.EventUpdate) {
	if m.queues[eventUpdate.Symbol] == nil { // FEEDBACK - Race condition not protected by . AddToQueue() and DeQueue() can be called concurrently.
		m.initQueue(eventUpdate.Symbol)
	}
	m.queues[eventUpdate.Symbol] <- eventUpdate
}

func (m *InQManager) DeQueue(symbol string) <-chan *dtos.EventUpdate { // FEEDBACK: change the name to Queue() or GetQueue() as DeQueue usually means removing an item from the queue.
	if m.queues[symbol] == nil {
		m.initQueue(symbol)
	}

	return m.queues[symbol]
}

func (m *InQManager) initQueue(currency string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.queues[currency] == nil {
		q := make(chan *dtos.EventUpdate, queueSize)
		m.queues[currency] = q
	}
}
