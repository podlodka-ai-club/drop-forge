package reviewrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orchv3/internal/agentmeta"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func setupChange(t *testing.T) (cloneDir, changePath string) {
	t.Helper()
	cloneDir = t.TempDir()
	changePath = filepath.Join("openspec", "changes", "x")
	root := filepath.Join(cloneDir, changePath)
	writeFile(t, filepath.Join(root, "proposal.md"), "# proposal\n")
	writeFile(t, filepath.Join(root, "design.md"), "# design\n")
	writeFile(t, filepath.Join(root, "tasks.md"), "# tasks\n")
	writeFile(t, filepath.Join(root, "specs", "cap", "spec.md"), "# spec\n")
	return cloneDir, changePath
}

func TestCollectTargetsForProposalReadsAllChangeFiles(t *testing.T) {
	cloneDir, changePath := setupChange(t)

	targets, err := CollectTargets(TargetInput{
		Stage:      agentmeta.StageProposal,
		CloneDir:   cloneDir,
		ChangePath: changePath,
	})
	if err != nil {
		t.Fatalf("CollectTargets: %v", err)
	}

	want := map[string]bool{
		"openspec/changes/x/proposal.md":       false,
		"openspec/changes/x/design.md":         false,
		"openspec/changes/x/tasks.md":          false,
		"openspec/changes/x/specs/cap/spec.md": false,
	}
	for _, tg := range targets {
		if _, ok := want[tg.Path]; ok {
			want[tg.Path] = true
		}
	}
	for path, found := range want {
		if !found {
			t.Errorf("expected target %q in result; got %v", path, targets)
		}
	}
}

func TestCollectTargetsTruncatesAndMarksWhenOverBudget(t *testing.T) {
	cloneDir := t.TempDir()
	changePath := filepath.Join("openspec", "changes", "x")
	root := filepath.Join(cloneDir, changePath)
	writeFile(t, filepath.Join(root, "a.md"), strings.Repeat("a", 60))
	writeFile(t, filepath.Join(root, "b.md"), strings.Repeat("b", 60))

	const maxBytes = 80
	targets, err := CollectTargets(TargetInput{
		Stage:      agentmeta.StageProposal,
		CloneDir:   cloneDir,
		ChangePath: changePath,
		MaxBytes:   maxBytes,
	})
	if err != nil {
		t.Fatalf("CollectTargets: %v", err)
	}

	totalLen := 0
	anyTruncated := false
	for _, tg := range targets {
		totalLen += len(tg.Content)
		if tg.Truncated {
			anyTruncated = true
		}
	}
	if totalLen > maxBytes {
		t.Errorf("total content length %d exceeded MaxBytes %d", totalLen, maxBytes)
	}
	if !anyTruncated {
		t.Errorf("expected at least one truncated target; got %+v", targets)
	}
}

func TestCollectTargetsForApplyIncludesDiffAndChangeContext(t *testing.T) {
	cloneDir, changePath := setupChange(t)
	const diff = "@@ diff body @@"

	targets, err := CollectTargets(TargetInput{
		Stage:      agentmeta.StageApply,
		CloneDir:   cloneDir,
		ChangePath: changePath,
		Diff:       diff,
	})
	if err != nil {
		t.Fatalf("CollectTargets: %v", err)
	}

	if len(targets) < 2 {
		t.Fatalf("expected diff plus change files; got %d targets", len(targets))
	}
	if targets[0].Path != "<diff>" {
		t.Errorf("expected first target Path=<diff>; got %q", targets[0].Path)
	}
	if targets[0].Content != diff {
		t.Errorf("expected first target Content=%q; got %q", diff, targets[0].Content)
	}

	foundChangeFile := false
	for _, tg := range targets[1:] {
		if strings.HasPrefix(tg.Path, "openspec/changes/x/") {
			foundChangeFile = true
			break
		}
	}
	if !foundChangeFile {
		t.Errorf("expected change files after diff target; got %+v", targets)
	}
}

func TestCollectTargetsForArchiveIncludesDiffAndArchivedFiles(t *testing.T) {
	cloneDir, changePath := setupChange(t)
	const diff = "@@ archive diff @@"

	targets, err := CollectTargets(TargetInput{
		Stage:      agentmeta.StageArchive,
		CloneDir:   cloneDir,
		ChangePath: changePath,
		Diff:       diff,
	})
	if err != nil {
		t.Fatalf("CollectTargets: %v", err)
	}

	if len(targets) < 2 {
		t.Fatalf("expected diff plus archived files; got %d targets", len(targets))
	}
	if targets[0].Path != "<diff>" {
		t.Errorf("expected first target Path=<diff>; got %q", targets[0].Path)
	}
	if targets[0].Content != diff {
		t.Errorf("expected first target Content=%q; got %q", diff, targets[0].Content)
	}

	foundChangeFile := false
	for _, tg := range targets[1:] {
		if strings.HasPrefix(tg.Path, "openspec/changes/x/") {
			foundChangeFile = true
			break
		}
	}
	if !foundChangeFile {
		t.Errorf("expected archived change files after diff target; got %+v", targets)
	}
}

func TestCollectTargetsForUnknownStageReturnsError(t *testing.T) {
	_, err := CollectTargets(TargetInput{
		Stage:    agentmeta.Stage("bogus"),
		CloneDir: t.TempDir(),
	})
	if err == nil {
		t.Fatalf("expected error for unknown stage")
	}
}

func TestCollectTargetsForProposalRequiresChangePath(t *testing.T) {
	_, err := CollectTargets(TargetInput{
		Stage:    agentmeta.StageProposal,
		CloneDir: t.TempDir(),
	})
	if err == nil {
		t.Fatalf("expected error when ChangePath is empty")
	}
}
