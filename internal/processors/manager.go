package processors

import (
	"encoding/json"
	"log/slog"
	"ob-manager/internal/dtos"
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
	proc := NewProcessor(currency, m.OutQ, m.DeQueuer)
	m.processors[currency] = proc
	proc.startProcessor()
}

func (m *Manager) Processor(currency string) *Processor {
	return m.processors[currency]
}

// GetOrderBook parses order book to a JSON to send to the subscriber.
func (m *Manager) GetOrderBook(curr string) []byte {
	proc := m.processors[curr]
	jsonStr, err := json.Marshal(proc.OrderBook())

	if err != nil {
		slog.Error("error on parsing order book to json", "Err", err)
	}

	return jsonStr
}

// UpdateBids updates the bids from the snapshot.
func (m *Manager) UpdateBids(currency string, bids map[float64]float64) {
	m.processors[currency].updateBids(bids)
}

// UpdateAsks updates the asks from the snapshot.
func (m *Manager) UpdateAsks(currency string, asks map[float64]float64) {
	m.processors[currency].updateAsks(asks)
}
