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
	m.initQueue(eventUpdate.Symbol)
	m.queues[eventUpdate.Symbol] <- eventUpdate
}

func (m *InQManager) DeQueue(symbol string) <-chan *dtos.EventUpdate {
	m.initQueue(symbol)

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
