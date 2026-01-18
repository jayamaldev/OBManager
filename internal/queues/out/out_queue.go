package outqueues

import "ob-manager/internal/dtos"

type Queue struct {
	q chan *dtos.EventUpdate
}

func NewQueue() *Queue {
	queueSize := 40000

	return &Queue{
		q: make(chan *dtos.EventUpdate, queueSize),
	}
}

func (q *Queue) AddToOutQ(event *dtos.EventUpdate) {
	q.q <- event
}

func (q *Queue) OutQ() <-chan *dtos.EventUpdate {
	return q.q
}
