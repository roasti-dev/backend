package events

import (
	"context"
	"log/slog"
)

type EventBus struct {
	ch       chan Event
	handlers []func(Event)
}

func NewBus(bufSize int) *EventBus {
	return &EventBus{ch: make(chan Event, bufSize)}
}

func (b *EventBus) Subscribe(handler func(Event)) {
	b.handlers = append(b.handlers, handler)
}

func (b *EventBus) Publish(e Event) {
	select {
	case b.ch <- e:
	default:
		slog.Warn("event bus buffer full, dropping event")
	}
}

func (b *EventBus) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case e := <-b.ch:
				for _, h := range b.handlers {
					h(e)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
