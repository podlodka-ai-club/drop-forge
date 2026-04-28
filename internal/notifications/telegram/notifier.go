package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"orchv3/internal/config"
	"orchv3/internal/events"
)

type Notifier struct {
	Config config.TelegramConfig
	Client *http.Client
}

func NewNotifier(cfg config.TelegramConfig) *Notifier {
	return &Notifier{
		Config: cfg,
		Client: &http.Client{Timeout: cfg.Timeout},
	}
}

func (notifier *Notifier) HandleEvent(ctx context.Context, event events.Event) error {
	if event.Type != events.TaskStatusChangedType {
		return nil
	}

	payload, ok := event.Payload.(events.TaskStatusChanged)
	if !ok {
		return fmt.Errorf("send telegram message: unexpected payload type %T", event.Payload)
	}

	return notifier.SendStatusChanged(ctx, payload)
}

func (notifier *Notifier) SendStatusChanged(ctx context.Context, payload events.TaskStatusChanged) error {
	message := formatStatusChangedMessage(payload)
	body, err := json.Marshal(sendMessageRequest{
		ChatID: strings.TrimSpace(notifier.Config.ChatID),
		Text:   message,
	})
	if err != nil {
		return fmt.Errorf("send telegram message: encode request: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", strings.TrimRight(notifier.Config.APIURL, "/"), notifier.Config.BotToken)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send telegram message: build request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	client := notifier.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("send telegram message: read response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("send telegram message: telegram api status %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var result sendMessageResponse
	if len(bytes.TrimSpace(responseBody)) > 0 {
		if err := json.Unmarshal(responseBody, &result); err != nil {
			return fmt.Errorf("send telegram message: decode response: %w", err)
		}
		if !result.OK {
			return fmt.Errorf("send telegram message: telegram api returned ok=false")
		}
	}

	return nil
}

func formatStatusChangedMessage(payload events.TaskStatusChanged) string {
	task := payloadOrFallback(payload.TaskIdentifier, payload.TaskID)
	target := payloadOrFallback(payload.TargetStateName, payload.TargetStateID)

	var builder strings.Builder
	builder.WriteString("Task status changed\n")
	builder.WriteString("Task: ")
	builder.WriteString(task)
	if strings.TrimSpace(payload.TaskTitle) != "" {
		builder.WriteString("\nTitle: ")
		builder.WriteString(strings.TrimSpace(payload.TaskTitle))
	}
	builder.WriteString("\nTarget state: ")
	builder.WriteString(target)

	return builder.String()
}

func payloadOrFallback(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}

	return strings.TrimSpace(fallback)
}

type sendMessageRequest struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

type sendMessageResponse struct {
	OK bool `json:"ok"`
}
