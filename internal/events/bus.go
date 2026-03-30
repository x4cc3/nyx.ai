package events

import (
	"sync"

	"nyx/internal/domain"
)

type Bus struct {
	mu          sync.Mutex
	subscribers map[string]map[chan domain.Event]struct{}
}

func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[string]map[chan domain.Event]struct{}),
	}
}

func (b *Bus) Publish(event domain.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for ch := range b.subscribers[event.FlowID] {
		select {
		case ch <- event:
		default:
		}
	}
}

func (b *Bus) Subscribe(flowID string) (<-chan domain.Event, func()) {
	ch := make(chan domain.Event, 16)

	b.mu.Lock()
	if b.subscribers[flowID] == nil {
		b.subscribers[flowID] = make(map[chan domain.Event]struct{})
	}
	b.subscribers[flowID][ch] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			delete(b.subscribers[flowID], ch)
			close(ch)
			if len(b.subscribers[flowID]) == 0 {
				delete(b.subscribers, flowID)
			}
		})
	}

	return ch, cancel
}
