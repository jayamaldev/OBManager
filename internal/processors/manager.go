package processors

import (
	"encoding/json"
	"log/slog"
	"ob-manager/internal/dtos"
	"sync"
)

type DeQueuer interface {
	DeQueue(symbol string) <-chan *dtos.EventUpdate
}

type OutQ interface {
	AddToOutQ(event *dtos.EventUpdate)
}

type Manager struct {
	DeQueuer
	OutQ

	mu         sync.RWMutex
	processors map[string]*Processor
}

func NewManager(deQueuer DeQueuer, outQ OutQ) *Manager {
	return &Manager{
		DeQueuer:   deQueuer,
		OutQ:       outQ,
		processors: make(map[string]*Processor),
	}
}

func (m *Manager) StartProcessor(currency string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	proc := NewProcessor(currency, m.OutQ, m.DeQueuer)
	m.processors[currency] = proc

	go proc.startProcessor()
}

func (m *Manager) Processor(currency string) *Processor {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.processors[currency]
}

// GetOrderBook parses order book to a JSON to send to the subscriber.
func (m *Manager) GetOrderBook(curr string) ([]byte, int) {
	proc := m.Processor(curr)
	ob := proc.OrderBook()
	lastUpdateId := ob.lastUpdateId

	jsonStr, err := json.Marshal(ob)

	if err != nil {
		slog.Error("error on parsing order book to json", "Err", err)
	}

	return jsonStr, lastUpdateId
}

// UpdateBids updates the bids from the snapshot.
func (m *Manager) UpdateBids(currency string, bids map[float64]float64) {
	m.Processor(currency).updateBids(bids)
}

// UpdateAsks updates the asks from the snapshot.
func (m *Manager) UpdateAsks(currency string, asks map[float64]float64) {
	m.Processor(currency).updateAsks(asks)
}

// SetOrderBookReady marks order book is populated and ready to process push events.
func (m *Manager) SetOrderBookReady(currency string, lastUpdateId int) {
	m.Processor(currency).SetReady(lastUpdateId)
}

// ResetProcessors clears all order books and prepare for a reconnection.
func (m *Manager) ResetProcessors() {
	for _, p := range m.processors {
		p.stopProcessor()
		slog.Info("Processor Stopped.", "Currency", p.currency)
	}

	clear(m.processors)
	slog.Info("Processors reset and Order Books Cleared")
}
