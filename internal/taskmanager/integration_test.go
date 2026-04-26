package taskmanager

import (
	"context"
	"os"
	"testing"

	"orchv3/internal/config"
)

func TestIntegrationGetTasksFromLinear(t *testing.T) {
	if os.Getenv("RUN_LINEAR_INTEGRATION") != "1" {
		t.Skip("set RUN_LINEAR_INTEGRATION=1 to run real Linear integration test")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() returned error: %v", err)
	}
	if err := cfg.TaskManager.Validate(); err != nil {
		t.Fatalf("TaskManager config invalid: %v", err)
	}

	manager := New(cfg.TaskManager)
	manager.LogWriter = os.Stdout

	tasks, err := manager.GetTasks(context.Background())
	if err != nil {
		t.Fatalf("GetTasks() returned error: %v", err)
	}
	if len(tasks) == 0 {
		t.Fatalf("GetTasks() returned 0 tasks; create at least one Linear issue in project %s with one of the managed state IDs", cfg.TaskManager.ProjectID)
	}

	for _, task := range tasks {
		t.Logf(
			"task id=%s identifier=%s state_id=%s state_name=%s comments=%d title=%q description_len=%d",
			task.ID,
			task.Identifier,
			task.State.ID,
			task.State.Name,
			len(task.Comments),
			task.Title,
			len(task.Description),
		)
	}
}
