package taskmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"orchv3/internal/config"
	"orchv3/internal/events"
)

func TestManagerGetTasksLogsProjectAndStateContext(t *testing.T) {
	var logs bytes.Buffer
	manager := &Manager{
		Config: validConfig(),
		Client: fakeClient{tasks: []Task{{
			ID:           "issue-1",
			Identifier:   "ENG-1",
			PullRequests: []PullRequest{{URL: "https://github.com/example/repo/pull/1"}},
		}}},
		LogWriter: &logs,
	}

	tasks, err := manager.GetTasks(context.Background())
	if err != nil {
		t.Fatalf("GetTasks() returned error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("tasks len = %d, want 1", len(tasks))
	}
	if len(tasks[0].PullRequests) != 1 || tasks[0].PullRequests[0].URL != "https://github.com/example/repo/pull/1" {
		t.Fatalf("pull requests = %#v", tasks[0].PullRequests)
	}

	events := decodeTaskManagerEvents(t, logs.String())
	assertTaskManagerLog(t, events, "taskmanager", "get tasks project=project-123 state_ids=state-propose,state-code,state-archive")
	assertTaskManagerLog(t, events, "taskmanager", "loaded 1 managed tasks for project project-123")
}

func TestGetTasksIncludesAIReviewStateIDsWhenConfigured(t *testing.T) {
	cfg := validConfig()
	cfg.NeedProposalAIReviewStateID = "state-proposal-ai-review"
	cfg.NeedCodeAIReviewStateID = "state-code-ai-review"
	cfg.NeedArchiveAIReviewStateID = "state-archive-ai-review"

	fake := &recordingClient{}
	manager := &Manager{
		Config: cfg,
		Client: fake,
	}

	if _, err := manager.GetTasks(context.Background()); err != nil {
		t.Fatalf("GetTasks() returned error: %v", err)
	}

	wantAIStates := []string{
		cfg.NeedProposalAIReviewStateID,
		cfg.NeedCodeAIReviewStateID,
		cfg.NeedArchiveAIReviewStateID,
	}
	for _, want := range wantAIStates {
		if !containsString(fake.getTasksStateIDs, want) {
			t.Fatalf("GetTasks stateIDs = %v, missing AI-review state %q", fake.getTasksStateIDs, want)
		}
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestManagerMoveTaskWrapsErrorAndLogs(t *testing.T) {
	var logs bytes.Buffer
	manager := &Manager{
		Config:    validConfig(),
		Client:    fakeClient{moveErr: errors.New("boom")},
		LogWriter: &logs,
	}

	err := manager.MoveTask(context.Background(), "issue-1", "state-2")
	if err == nil {
		t.Fatal("MoveTask() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "move task issue-1 to state state-2") {
		t.Fatalf("MoveTask() error = %q", err.Error())
	}

	events := decodeTaskManagerEvents(t, logs.String())
	assertTaskManagerLog(t, events, "taskmanager", "move task=issue-1 state=state-2")
	assertTaskManagerLog(t, events, "taskmanager", "move task=issue-1 state=state-2: boom")
}

func TestManagerMoveTaskPublishesStatusChangedEvent(t *testing.T) {
	publisher := &recordingPublisher{}
	manager := &Manager{
		Config:    validConfig(),
		Client:    fakeClient{},
		Publisher: publisher,
	}

	if err := manager.MoveTask(context.Background(), "issue-1", "state-2"); err != nil {
		t.Fatalf("MoveTask() returned error: %v", err)
	}

	if len(publisher.events) != 1 {
		t.Fatalf("published events len = %d, want 1", len(publisher.events))
	}
	event := publisher.events[0]
	if event.Type != events.TaskStatusChangedType {
		t.Fatalf("event type = %q, want %q", event.Type, events.TaskStatusChangedType)
	}
	if event.OccurredAt.IsZero() {
		t.Fatal("event occurred at is zero")
	}
	payload, ok := event.Payload.(events.TaskStatusChanged)
	if !ok {
		t.Fatalf("payload type = %T, want TaskStatusChanged", event.Payload)
	}
	if payload.TaskID != "issue-1" || payload.TargetStateID != "state-2" {
		t.Fatalf("payload = %#v", payload)
	}
	if payload.OccurredAt.IsZero() {
		t.Fatal("payload occurred at is zero")
	}
}

func TestManagerMoveTaskWithContextPublishesExpandedStatusChangedEvent(t *testing.T) {
	publisher := &recordingPublisher{}
	manager := &Manager{
		Config:    validConfig(),
		Client:    fakeClient{},
		Publisher: publisher,
	}

	err := manager.MoveTaskWithContext(context.Background(), "issue-1", "state-review", StatusChangeContext{
		TaskIdentifier:    "DRO-50",
		TaskTitle:         "Доработать формат сообщения в TG",
		SourceStateID:     "state-progress",
		SourceStateName:   "Code in Progress",
		TargetStateName:   "Need Code Review",
		PullRequestURL:    "https://github.com/example/repo/pull/50",
		PullRequestBranch: "codex/proposal/dro-50",
	})
	if err != nil {
		t.Fatalf("MoveTaskWithContext() returned error: %v", err)
	}

	if len(publisher.events) != 1 {
		t.Fatalf("published events len = %d, want 1", len(publisher.events))
	}
	payload, ok := publisher.events[0].Payload.(events.TaskStatusChanged)
	if !ok {
		t.Fatalf("payload type = %T, want TaskStatusChanged", publisher.events[0].Payload)
	}
	if payload.TaskID != "issue-1" || payload.TargetStateID != "state-review" {
		t.Fatalf("payload required fields = %#v", payload)
	}
	if payload.TaskIdentifier != "DRO-50" || payload.TaskTitle != "Доработать формат сообщения в TG" {
		t.Fatalf("task context = %#v", payload)
	}
	if payload.SourceStateID != "state-progress" || payload.SourceStateName != "Code in Progress" || payload.TargetStateName != "Need Code Review" {
		t.Fatalf("state context = %#v", payload)
	}
	if payload.PullRequestURL != "https://github.com/example/repo/pull/50" || payload.PullRequestBranch != "codex/proposal/dro-50" {
		t.Fatalf("pr context = %#v", payload)
	}
}

func TestManagerMoveTaskDoesNotPublishWhenLinearMoveFails(t *testing.T) {
	publisher := &recordingPublisher{}
	manager := &Manager{
		Config:    validConfig(),
		Client:    fakeClient{moveErr: errors.New("linear down")},
		Publisher: publisher,
	}

	err := manager.MoveTask(context.Background(), "issue-1", "state-2")
	if err == nil {
		t.Fatal("MoveTask() error = nil, want non-nil")
	}
	if len(publisher.events) != 0 {
		t.Fatalf("published events len = %d, want 0", len(publisher.events))
	}
}

func TestManagerMoveTaskKeepsSuccessWhenPublisherFails(t *testing.T) {
	var logs bytes.Buffer
	manager := &Manager{
		Config:    validConfig(),
		Client:    fakeClient{},
		LogWriter: &logs,
		Publisher: &recordingPublisher{err: errors.New("telegram failed")},
	}

	if err := manager.MoveTask(context.Background(), "issue-1", "state-2"); err != nil {
		t.Fatalf("MoveTask() returned error: %v", err)
	}

	events := decodeTaskManagerEvents(t, logs.String())
	assertTaskManagerLog(t, events, "taskmanager", "publish event=task.status_changed task=issue-1 state=state-2: telegram failed")
}

func TestManagerMoveTaskSucceedsWithoutPublisher(t *testing.T) {
	manager := &Manager{
		Config: validConfig(),
		Client: fakeClient{},
	}

	if err := manager.MoveTask(context.Background(), "issue-1", "state-2"); err != nil {
		t.Fatalf("MoveTask() returned error: %v", err)
	}
}

func TestManagerMoveTaskUsesConfiguredReviewStateIDs(t *testing.T) {
	tests := []struct {
		name    string
		stateID func(config.LinearTaskManagerConfig) string
	}{
		{
			name: "proposal review",
			stateID: func(cfg config.LinearTaskManagerConfig) string {
				return cfg.NeedProposalReviewStateID
			},
		},
		{
			name: "code review",
			stateID: func(cfg config.LinearTaskManagerConfig) string {
				return cfg.NeedCodeReviewStateID
			},
		},
		{
			name: "archive review",
			stateID: func(cfg config.LinearTaskManagerConfig) string {
				return cfg.NeedArchiveReviewStateID
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &recordingClient{}
			cfg := validConfig()
			manager := &Manager{
				Config: cfg,
				Client: fake,
			}

			targetStateID := tt.stateID(cfg)
			if err := manager.MoveTask(context.Background(), "issue-1", targetStateID); err != nil {
				t.Fatalf("MoveTask() returned error: %v", err)
			}

			if fake.moveTaskID != "issue-1" {
				t.Fatalf("moveTaskID = %q, want %q", fake.moveTaskID, "issue-1")
			}
			if fake.moveStateID != targetStateID {
				t.Fatalf("moveStateID = %q, want %q", fake.moveStateID, targetStateID)
			}
		})
	}
}

func TestManagerMoveTaskUsesConfiguredInProgressStateIDs(t *testing.T) {
	tests := []struct {
		name    string
		stateID func(config.LinearTaskManagerConfig) string
	}{
		{
			name: "proposing in progress",
			stateID: func(cfg config.LinearTaskManagerConfig) string {
				return cfg.ProposingInProgressStateID
			},
		},
		{
			name: "code in progress",
			stateID: func(cfg config.LinearTaskManagerConfig) string {
				return cfg.CodeInProgressStateID
			},
		},
		{
			name: "archiving in progress",
			stateID: func(cfg config.LinearTaskManagerConfig) string {
				return cfg.ArchivingInProgressStateID
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &recordingClient{}
			cfg := validConfig()
			manager := &Manager{
				Config: cfg,
				Client: fake,
			}

			targetStateID := tt.stateID(cfg)
			if err := manager.MoveTask(context.Background(), "issue-1", targetStateID); err != nil {
				t.Fatalf("MoveTask() returned error: %v", err)
			}

			if fake.moveTaskID != "issue-1" {
				t.Fatalf("moveTaskID = %q, want %q", fake.moveTaskID, "issue-1")
			}
			if fake.moveStateID != targetStateID {
				t.Fatalf("moveStateID = %q, want %q", fake.moveStateID, targetStateID)
			}
		})
	}
}

func TestManagerAddPRRejectsInvalidURL(t *testing.T) {
	manager := &Manager{
		Config: validConfig(),
		Client: fakeClient{},
	}

	err := manager.AddPR(context.Background(), "issue-1", "not-a-url")
	if err == nil {
		t.Fatal("AddPR() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "valid absolute url") {
		t.Fatalf("AddPR() error = %q", err.Error())
	}
}

func TestManagerValidatesConfigBeforeClientCall(t *testing.T) {
	manager := &Manager{
		Config: config.LinearTaskManagerConfig{},
		Client: fakeClient{},
	}

	err := manager.AddComment(context.Background(), "issue-1", "body")
	if err == nil {
		t.Fatal("AddComment() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "LINEAR_") {
		t.Fatalf("AddComment() error = %q", err.Error())
	}
}

type fakeClient struct {
	tasks       []Task
	getTasksErr error
	moveErr     error
	commentErr  error
	addPRErr    error
}

func (client fakeClient) GetTasks(ctx context.Context, projectID string, stateIDs []string) ([]Task, error) {
	return client.tasks, client.getTasksErr
}

func (client fakeClient) MoveTask(ctx context.Context, taskID string, stateID string) error {
	return client.moveErr
}

func (client fakeClient) AddComment(ctx context.Context, taskID string, body string) error {
	return client.commentErr
}

func (client fakeClient) AddPR(ctx context.Context, taskID string, prURL string) error {
	return client.addPRErr
}

type recordingClient struct {
	moveTaskID       string
	moveStateID      string
	getTasksStateIDs []string
}

type recordingPublisher struct {
	events []events.Event
	err    error
}

func (publisher *recordingPublisher) Publish(ctx context.Context, event events.Event) error {
	publisher.events = append(publisher.events, event)
	if payload, ok := event.Payload.(events.TaskStatusChanged); ok && payload.OccurredAt.After(time.Now().Add(time.Minute)) {
		return errors.New("unexpected future event")
	}

	return publisher.err
}

func (client *recordingClient) GetTasks(ctx context.Context, projectID string, stateIDs []string) ([]Task, error) {
	client.getTasksStateIDs = append([]string(nil), stateIDs...)
	return nil, nil
}

func (client *recordingClient) MoveTask(ctx context.Context, taskID string, stateID string) error {
	client.moveTaskID = taskID
	client.moveStateID = stateID
	return nil
}

func (client *recordingClient) AddComment(ctx context.Context, taskID string, body string) error {
	return nil
}

func (client *recordingClient) AddPR(ctx context.Context, taskID string, prURL string) error {
	return nil
}

type taskManagerEvent struct {
	Module  string `json:"module"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

func decodeTaskManagerEvents(t *testing.T, output string) []taskManagerEvent {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	events := make([]taskManagerEvent, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var event taskManagerEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("decode log event %q: %v", line, err)
		}
		events = append(events, event)
	}

	return events
}

func assertTaskManagerLog(t *testing.T, events []taskManagerEvent, module string, message string) {
	t.Helper()

	for _, event := range events {
		if event.Module == module && strings.Contains(event.Message, message) {
			return
		}
	}

	t.Fatalf("missing log module=%q message containing %q in %#v", module, message, events)
}

func validConfig() config.LinearTaskManagerConfig {
	return config.LinearTaskManagerConfig{
		APIURL:                     "https://api.linear.app/graphql",
		APIToken:                   "linear-token",
		ProjectID:                  "project-123",
		ReadyToProposeStateID:      "state-propose",
		ReadyToCodeStateID:         "state-code",
		ReadyToArchiveStateID:      "state-archive",
		ProposingInProgressStateID: "state-proposing-progress",
		CodeInProgressStateID:      "state-code-progress",
		ArchivingInProgressStateID: "state-archiving-progress",
		NeedProposalReviewStateID:  "state-proposal-review",
		NeedCodeReviewStateID:      "state-code-review",
		NeedArchiveReviewStateID:   "state-archive-review",
	}
}
