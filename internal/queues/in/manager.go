package inqueues

import (
	"ob-manager/internal/dtos"
	"sync"
)

const (
	queueSize = 10000
)

type InQManager struct {
	mu     sync.RWMutex
	queues map[string]chan *dtos.EventUpdate
}

func NewQManager() *InQManager {
	return &InQManager{
		queues: make(map[string]chan *dtos.EventUpdate),
	}
}

func (m *InQManager) AddToQueue(eventUpdate *dtos.EventUpdate) {
	q := m.getOrCreateQueue(eventUpdate.Symbol)
	q <- eventUpdate
}

func (m *InQManager) Queue(symbol string) <-chan *dtos.EventUpdate {
	return m.getOrCreateQueue(symbol)
}

func (m *InQManager) getOrCreateQueue(currency string) chan *dtos.EventUpdate {
	m.mu.RLock() // read lock

	if q, ok := m.queues[currency]; ok {
		m.mu.RUnlock()

		return q
	}

	m.mu.RUnlock() //unlock read lock

	m.mu.Lock() // write lock to make a new channel
	defer m.mu.Unlock()

	// double-check this to confirm another goroutine is not created
	if q, ok := m.queues[currency]; ok {
		return q
	}

	q := make(chan *dtos.EventUpdate, queueSize)
	m.queues[currency] = q

	return q
}
