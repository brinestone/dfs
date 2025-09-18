package events

import (
	"context"
	"sync"
)

type EventTarget string

type Event struct {
	Target EventTarget
	Id     string
	Data   map[string]any
}

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[EventTarget][]chan Event
}

func NewBus(ctx context.Context) *EventBus {
	publishCh := make(chan Event)
	go func() {
		<-ctx.Done()
		close(publishCh)
	}()
	return &EventBus{
		mu:          sync.RWMutex{},
		subscribers: make(map[EventTarget][]chan Event),
	}
}

func (d *EventBus) Publish(e Event) {
}

func (e EventTarget) Matches(t string) bool {
	return t == string(e)
}

func (d *EventBus) unsubscribe(i int, target EventTarget) {
	d.mu.Lock()
	defer d.mu.Unlock()

	list := d.subscribers[target]
	d.subscribers[target] = append(list[:i], list[i+1:]...)
	if len(d.subscribers[target]) == 0 {
		delete(d.subscribers, target)
	}
}

func (d *EventBus) Subscribe(ctx context.Context, target EventTarget) <-chan Event {
	d.mu.Lock()
	defer d.mu.Unlock()

	ch := make(chan Event, 10)
	d.subscribers[target] = append(d.subscribers[target], ch)
	chIndex := len(d.subscribers[target]) - 1

	go func() {
		<-ctx.Done()
		d.unsubscribe(chIndex, target)
	}()
	return ch
}
