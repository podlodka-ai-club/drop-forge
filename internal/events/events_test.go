package events

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestDispatcherPublishesMatchingEvent(t *testing.T) {
	dispatcher := NewDispatcher()
	var got Event
	dispatcher.Subscribe(TaskStatusChangedType, HandlerFunc(func(ctx context.Context, event Event) error {
		got = event
		return nil
	}))

	want := Event{
		Type:       TaskStatusChangedType,
		OccurredAt: time.Now(),
		Payload:    TaskStatusChanged{TaskID: "task-1", TargetStateID: "state-1"},
	}
	if err := dispatcher.Publish(context.Background(), want); err != nil {
		t.Fatalf("Publish() returned error: %v", err)
	}

	if got.Type != want.Type {
		t.Fatalf("event type = %q, want %q", got.Type, want.Type)
	}
	payload, ok := got.Payload.(TaskStatusChanged)
	if !ok {
		t.Fatalf("payload type = %T, want TaskStatusChanged", got.Payload)
	}
	if payload.TaskID != "task-1" || payload.TargetStateID != "state-1" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestTaskStatusChangedSupportsOptionalPRContext(t *testing.T) {
	payload := TaskStatusChanged{
		TaskID:            "task-1",
		TargetStateID:     "state-review",
		TaskIdentifier:    "DRO-50",
		TaskTitle:         "Доработать формат сообщения в TG",
		TargetStateName:   "Need Code Review",
		PullRequestURL:    "https://github.com/example/repo/pull/50",
		PullRequestBranch: "codex/proposal/dro-50",
	}

	event := Event{
		Type:    TaskStatusChangedType,
		Payload: payload,
	}
	if event.Type != TaskStatusChangedType {
		t.Fatalf("event type = %q, want %q", event.Type, TaskStatusChangedType)
	}
	got, ok := event.Payload.(TaskStatusChanged)
	if !ok {
		t.Fatalf("payload type = %T, want TaskStatusChanged", event.Payload)
	}
	if got.PullRequestURL != "https://github.com/example/repo/pull/50" || got.PullRequestBranch != "codex/proposal/dro-50" {
		t.Fatalf("pr context = url %q branch %q", got.PullRequestURL, got.PullRequestBranch)
	}
	if got.TaskID != "task-1" || got.TargetStateID != "state-review" {
		t.Fatalf("required fields changed: %#v", got)
	}
}

func TestDispatcherSkipsUnrelatedEvent(t *testing.T) {
	dispatcher := NewDispatcher()
	called := false
	dispatcher.Subscribe(TaskStatusChangedType, HandlerFunc(func(ctx context.Context, event Event) error {
		called = true
		return nil
	}))

	if err := dispatcher.Publish(context.Background(), Event{Type: "other.event"}); err != nil {
		t.Fatalf("Publish() returned error: %v", err)
	}

	if called {
		t.Fatal("subscriber was called for unrelated event")
	}
}

func TestDispatcherCallsMultipleSubscribers(t *testing.T) {
	dispatcher := NewDispatcher()
	var calls int
	dispatcher.Subscribe(TaskStatusChangedType, HandlerFunc(func(ctx context.Context, event Event) error {
		calls++
		return nil
	}))
	dispatcher.Subscribe(TaskStatusChangedType, HandlerFunc(func(ctx context.Context, event Event) error {
		calls++
		return nil
	}))

	if err := dispatcher.Publish(context.Background(), Event{Type: TaskStatusChangedType}); err != nil {
		t.Fatalf("Publish() returned error: %v", err)
	}

	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestDispatcherReturnsSubscriberError(t *testing.T) {
	dispatcher := NewDispatcher()
	dispatcher.Subscribe(TaskStatusChangedType, HandlerFunc(func(ctx context.Context, event Event) error {
		return errors.New("delivery failed")
	}))

	err := dispatcher.Publish(context.Background(), Event{Type: TaskStatusChangedType})
	if err == nil {
		t.Fatal("Publish() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "handle event task.status_changed subscriber 0") {
		t.Fatalf("Publish() error = %q, want event context", err.Error())
	}
	if !strings.Contains(err.Error(), "delivery failed") {
		t.Fatalf("Publish() error = %q, want subscriber error", err.Error())
	}
}
