package taskmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"orchv3/internal/config"
)

func TestManagerGetTasksLogsProjectAndStateContext(t *testing.T) {
	var logs bytes.Buffer
	manager := &Manager{
		Config:    validConfig(),
		Client:    fakeClient{tasks: []Task{{ID: "issue-1", Identifier: "ENG-1"}}},
		LogWriter: &logs,
	}

	tasks, err := manager.GetTasks(context.Background())
	if err != nil {
		t.Fatalf("GetTasks() returned error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("tasks len = %d, want 1", len(tasks))
	}

	events := decodeTaskManagerEvents(t, logs.String())
	assertTaskManagerLog(t, events, "taskmanager", "get tasks project=project-123 state_ids=state-propose,state-code,state-archive")
	assertTaskManagerLog(t, events, "taskmanager", "loaded 1 managed tasks for project project-123")
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
	moveTaskID  string
	moveStateID string
}

func (client *recordingClient) GetTasks(ctx context.Context, projectID string, stateIDs []string) ([]Task, error) {
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
