package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"orchv3/internal/config"
	"orchv3/internal/events"
)

func TestNotifierSendsStatusChangedMessage(t *testing.T) {
	var gotPath string
	var gotRequest sendMessageRequest
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotPath = request.URL.Path
		if err := json.NewDecoder(request.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	notifier := NewNotifier(config.TelegramConfig{
		BotToken: "token-123",
		ChatID:   "chat-456",
		APIURL:   server.URL,
		Timeout:  time.Second,
	}, "state-code-review")

	err := notifier.HandleEvent(context.Background(), events.Event{
		Type: events.TaskStatusChangedType,
		Payload: events.TaskStatusChanged{
			TaskID:          "task-1",
			TargetStateID:   "state-code-review",
			TaskIdentifier:  "DRO-45",
			TaskTitle:       "Добавить Telegram уведомление",
			TargetStateName: "Need Code Review",
			PullRequestURL:  "https://github.com/example/repo/pull/45",
		},
	})
	if err != nil {
		t.Fatalf("HandleEvent() returned error: %v", err)
	}

	if gotPath != "/bottoken-123/sendMessage" {
		t.Fatalf("path = %q, want telegram sendMessage path", gotPath)
	}
	if gotRequest.ChatID != "chat-456" {
		t.Fatalf("chat_id = %q, want chat-456", gotRequest.ChatID)
	}
	for _, want := range []string{"DRO-45", "Добавить Telegram уведомление", "Need Code Review", "https://github.com/example/repo/pull/45"} {
		if !strings.Contains(gotRequest.Text, want) {
			t.Fatalf("text = %q, want substring %q", gotRequest.Text, want)
		}
	}
}

func TestNotifierSkipsNonReviewStatusChangedMessage(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls++
		t.Fatal("telegram API must not be called for non-review state")
	}))
	defer server.Close()

	notifier := NewNotifier(config.TelegramConfig{
		BotToken: "token-123",
		ChatID:   "chat-456",
		APIURL:   server.URL,
		Timeout:  time.Second,
	}, "state-code-review")

	err := notifier.HandleEvent(context.Background(), events.Event{
		Type: events.TaskStatusChangedType,
		Payload: events.TaskStatusChanged{
			TaskID:        "task-1",
			TargetStateID: "state-code-progress",
		},
	})
	if err != nil {
		t.Fatalf("HandleEvent() returned error: %v", err)
	}
	if calls != 0 {
		t.Fatalf("telegram calls = %d, want 0", calls)
	}
}

func TestNotifierFormatsBranchFallbackWhenPRURLMissing(t *testing.T) {
	gotText := formatStatusChangedMessage(events.TaskStatusChanged{
		TaskID:            "task-1",
		TargetStateID:     "state-code-review",
		TaskIdentifier:    "DRO-50",
		TaskTitle:         "Доработать формат сообщения в TG",
		TargetStateName:   "Need Code Review",
		PullRequestBranch: "codex/proposal/dro-50",
	})

	for _, want := range []string{"DRO-50", "Доработать формат сообщения в TG", "Need Code Review", "codex/proposal/dro-50"} {
		if !strings.Contains(gotText, want) {
			t.Fatalf("text = %q, want substring %q", gotText, want)
		}
	}
}

func TestNotifierFallsBackToStableIDs(t *testing.T) {
	gotText := formatStatusChangedMessage(events.TaskStatusChanged{
		TaskID:        "task-1",
		TargetStateID: "state-1",
	})

	if !strings.Contains(gotText, "task-1") {
		t.Fatalf("text = %q, want task id fallback", gotText)
	}
	if !strings.Contains(gotText, "state-1") {
		t.Fatalf("text = %q, want state id fallback", gotText)
	}
}

func TestNotifierReturnsErrorForTelegramFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusBadGateway)
		_, _ = writer.Write([]byte(`bad gateway`))
	}))
	defer server.Close()

	notifier := NewNotifier(config.TelegramConfig{
		BotToken: "token-123",
		ChatID:   "chat-456",
		APIURL:   server.URL,
		Timeout:  time.Second,
	}, "state-1")

	err := notifier.HandleEvent(context.Background(), events.Event{
		Type: events.TaskStatusChangedType,
		Payload: events.TaskStatusChanged{
			TaskID:        "task-1",
			TargetStateID: "state-1",
		},
	})
	if err == nil {
		t.Fatal("HandleEvent() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "send telegram message") {
		t.Fatalf("HandleEvent() error = %q, want delivery context", err.Error())
	}
}

func TestNotifierReturnsErrorForTelegramOKFalse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"ok":false}`))
	}))
	defer server.Close()

	notifier := NewNotifier(config.TelegramConfig{
		BotToken: "token-123",
		ChatID:   "chat-456",
		APIURL:   server.URL,
		Timeout:  time.Second,
	}, "state-1")

	err := notifier.HandleEvent(context.Background(), events.Event{
		Type: events.TaskStatusChangedType,
		Payload: events.TaskStatusChanged{
			TaskID:        "task-1",
			TargetStateID: "state-1",
		},
	})
	if err == nil {
		t.Fatal("HandleEvent() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "ok=false") {
		t.Fatalf("HandleEvent() error = %q, want ok=false context", err.Error())
	}
}

func TestNotifierReturnsErrorForTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(50 * time.Millisecond)
	}))
	defer server.Close()

	notifier := NewNotifier(config.TelegramConfig{
		BotToken: "token-123",
		ChatID:   "chat-456",
		APIURL:   server.URL,
		Timeout:  1 * time.Millisecond,
	}, "state-1")

	err := notifier.HandleEvent(context.Background(), events.Event{
		Type: events.TaskStatusChangedType,
		Payload: events.TaskStatusChanged{
			TaskID:        "task-1",
			TargetStateID: "state-1",
		},
	})
	if err == nil {
		t.Fatal("HandleEvent() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "send telegram message") {
		t.Fatalf("HandleEvent() error = %q, want delivery context", err.Error())
	}
}
