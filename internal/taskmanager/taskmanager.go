package taskmanager

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"orchv3/internal/config"
	"orchv3/internal/steplog"
)

type Task struct {
	ID           string
	Identifier   string
	Title        string
	Description  string
	ProjectID    string
	State        WorkflowState
	Comments     []Comment
	PullRequests []PullRequest
}

type PullRequest struct {
	URL    string
	Branch string
}

type WorkflowState struct {
	ID   string
	Name string
}

type Comment struct {
	ID        string
	Body      string
	CreatedAt time.Time
	User      User
}

type User struct {
	ID          string
	Name        string
	DisplayName string
	Email       string
}

type Client interface {
	GetTasks(ctx context.Context, projectID string, stateIDs []string) ([]Task, error)
	MoveTask(ctx context.Context, taskID string, stateID string) error
	AddComment(ctx context.Context, taskID string, body string) error
	AddPR(ctx context.Context, taskID string, prURL string) error
}

type Manager struct {
	Config    config.LinearTaskManagerConfig
	Client    Client
	LogWriter io.Writer
}

func New(cfg config.LinearTaskManagerConfig) *Manager {
	return &Manager{
		Config:    cfg,
		LogWriter: os.Stderr,
	}
}

func (manager *Manager) GetTasks(ctx context.Context) ([]Task, error) {
	if err := manager.Config.Validate(); err != nil {
		return nil, fmt.Errorf("validate task manager config: %w", err)
	}

	stateIDs := manager.Config.ManagedStateIDs()
	logger := steplog.New(writerOrDiscard(manager.LogWriter))
	logger.Infof("taskmanager", "get tasks project=%s state_ids=%s", manager.Config.ProjectID, strings.Join(stateIDs, ","))

	tasks, err := manager.client().GetTasks(ctx, manager.Config.ProjectID, stateIDs)
	if err != nil {
		logger.Errorf("taskmanager", "get tasks project=%s: %v", manager.Config.ProjectID, err)
		return nil, fmt.Errorf("get tasks for project %s: %w", manager.Config.ProjectID, err)
	}

	logger.Infof("taskmanager", "loaded %d managed tasks for project %s", len(tasks), manager.Config.ProjectID)
	return tasks, nil
}

func (manager *Manager) MoveTask(ctx context.Context, taskID string, stateID string) error {
	if err := manager.Config.Validate(); err != nil {
		return fmt.Errorf("validate task manager config: %w", err)
	}
	if err := validateRequiredField("task id", taskID); err != nil {
		return err
	}
	if err := validateRequiredField("state id", stateID); err != nil {
		return err
	}

	logger := steplog.New(writerOrDiscard(manager.LogWriter))
	logger.Infof("taskmanager", "move task=%s state=%s", taskID, stateID)
	if err := manager.client().MoveTask(ctx, taskID, stateID); err != nil {
		logger.Errorf("taskmanager", "move task=%s state=%s: %v", taskID, stateID, err)
		return fmt.Errorf("move task %s to state %s: %w", taskID, stateID, err)
	}

	logger.Infof("taskmanager", "moved task=%s state=%s", taskID, stateID)
	return nil
}

func (manager *Manager) AddComment(ctx context.Context, taskID string, body string) error {
	if err := manager.Config.Validate(); err != nil {
		return fmt.Errorf("validate task manager config: %w", err)
	}
	if err := validateRequiredField("task id", taskID); err != nil {
		return err
	}
	if err := validateRequiredField("comment body", body); err != nil {
		return err
	}

	logger := steplog.New(writerOrDiscard(manager.LogWriter))
	logger.Infof("taskmanager", "add comment task=%s", taskID)
	if err := manager.client().AddComment(ctx, taskID, body); err != nil {
		logger.Errorf("taskmanager", "add comment task=%s: %v", taskID, err)
		return fmt.Errorf("add comment to task %s: %w", taskID, err)
	}

	logger.Infof("taskmanager", "added comment task=%s", taskID)
	return nil
}

func (manager *Manager) AddPR(ctx context.Context, taskID string, prURL string) error {
	if err := manager.Config.Validate(); err != nil {
		return fmt.Errorf("validate task manager config: %w", err)
	}
	if err := validateRequiredField("task id", taskID); err != nil {
		return err
	}
	if err := validatePRURL(prURL); err != nil {
		return err
	}

	logger := steplog.New(writerOrDiscard(manager.LogWriter))
	logger.Infof("taskmanager", "add pr task=%s pr=%s", taskID, prURL)
	if err := manager.client().AddPR(ctx, taskID, prURL); err != nil {
		logger.Errorf("taskmanager", "add pr task=%s pr=%s: %v", taskID, prURL, err)
		return fmt.Errorf("add pr %s to task %s: %w", prURL, taskID, err)
	}

	logger.Infof("taskmanager", "added pr task=%s pr=%s", taskID, prURL)
	return nil
}

func (manager *Manager) client() Client {
	if manager.Client != nil {
		if client, ok := manager.Client.(*LinearClient); ok {
			client.LogWriter = writerOrDiscard(manager.LogWriter)
		}
		return manager.Client
	}

	client := NewLinearClient(manager.Config)
	client.LogWriter = writerOrDiscard(manager.LogWriter)
	manager.Client = client
	return manager.Client
}

func validateRequiredField(field string, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s must not be empty", field)
	}

	return nil
}

func validatePRURL(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("pr url must not be empty")
	}

	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("pr url must be a valid absolute url")
	}

	return nil
}

func writerOrDiscard(writer io.Writer) io.Writer {
	if writer == nil {
		return io.Discard
	}

	return writer
}
