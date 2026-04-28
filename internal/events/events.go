package events

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const TaskStatusChangedType = "task.status_changed"

type Event struct {
	Type       string
	OccurredAt time.Time
	Payload    any
}

type TaskStatusChanged struct {
	TaskID          string
	TargetStateID   string
	OccurredAt      time.Time
	TaskIdentifier  string
	TaskTitle       string
	SourceStateID   string
	SourceStateName string
	TargetStateName string
}

type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

type Handler interface {
	HandleEvent(ctx context.Context, event Event) error
}

type HandlerFunc func(ctx context.Context, event Event) error

func (fn HandlerFunc) HandleEvent(ctx context.Context, event Event) error {
	return fn(ctx, event)
}

type Dispatcher struct {
	handlers map[string][]Handler
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[string][]Handler),
	}
}

func (dispatcher *Dispatcher) Subscribe(eventType string, handler Handler) {
	if dispatcher.handlers == nil {
		dispatcher.handlers = make(map[string][]Handler)
	}
	if handler == nil {
		return
	}

	dispatcher.handlers[eventType] = append(dispatcher.handlers[eventType], handler)
}

func (dispatcher *Dispatcher) Publish(ctx context.Context, event Event) error {
	if dispatcher == nil {
		return nil
	}

	handlers := dispatcher.handlers[event.Type]
	var publishErr error
	for index, handler := range handlers {
		if err := handler.HandleEvent(ctx, event); err != nil {
			publishErr = errors.Join(publishErr, fmt.Errorf("handle event %s subscriber %d: %w", event.Type, index, err))
		}
	}

	return publishErr
}
