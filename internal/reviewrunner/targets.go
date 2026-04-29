package reviewrunner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"orchv3/internal/agentmeta"
)

// Target is one piece of context handed to the reviewer prompt. Path is a
// repository-relative path (or the literal "<diff>" for diff content). Content
// is the bytes that go into the prompt. Truncated is true when budget pressure
// caused this target to be cut short or dropped.
type Target struct {
	Path      string
	Content   string
	Truncated bool
}

// TargetInput drives CollectTargets.
type TargetInput struct {
	Stage      agentmeta.Stage
	CloneDir   string
	MaxBytes   int    // 0 means no budget enforcement
	ChangePath string // repo-relative path to the OpenSpec change directory
	Diff       string // git diff payload (Apply/Archive)
}

// CollectTargets dispatches by stage and returns the assembled target list.
func CollectTargets(in TargetInput) ([]Target, error) {
	switch in.Stage {
	case agentmeta.StageProposal:
		return collectProposal(in)
	case agentmeta.StageApply:
		return collectApply(in)
	case agentmeta.StageArchive:
		return collectArchive(in)
	}
	return nil, fmt.Errorf("unknown stage %s", in.Stage)
}

func collectProposal(in TargetInput) ([]Target, error) {
	if strings.TrimSpace(in.ChangePath) == "" {
		return nil, fmt.Errorf("ChangePath required for proposal stage")
	}
	files, err := walkMarkdownFiles(filepath.Join(in.CloneDir, in.ChangePath))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)

	out := make([]Target, 0, len(files))
	for _, abs := range files {
		rel, err := filepath.Rel(in.CloneDir, abs)
		if err != nil {
			return nil, fmt.Errorf("relativize target path: %w", err)
		}
		rel = filepath.ToSlash(rel)
		content, err := os.ReadFile(abs)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		out = append(out, Target{Path: rel, Content: string(content)})
	}
	return budget(out, in.MaxBytes), nil
}

func collectApply(in TargetInput) ([]Target, error) {
	out := []Target{{Path: "<diff>", Content: in.Diff}}
	if strings.TrimSpace(in.ChangePath) != "" {
		change, err := collectProposal(TargetInput{
			Stage:      agentmeta.StageProposal,
			CloneDir:   in.CloneDir,
			ChangePath: in.ChangePath,
			MaxBytes:   in.MaxBytes,
		})
		if err == nil {
			out = append(out, change...)
		}
	}
	return budget(out, in.MaxBytes), nil
}

func collectArchive(in TargetInput) ([]Target, error) {
	out := []Target{{Path: "<diff>", Content: in.Diff}}
	if strings.TrimSpace(in.ChangePath) != "" {
		change, err := collectProposal(TargetInput{
			Stage:      agentmeta.StageProposal,
			CloneDir:   in.CloneDir,
			ChangePath: in.ChangePath,
			MaxBytes:   in.MaxBytes,
		})
		if err == nil {
			out = append(out, change...)
		}
	}
	return budget(out, in.MaxBytes), nil
}

func walkMarkdownFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".md") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", root, err)
	}
	return files, nil
}

// budget enforces MaxBytes by truncating Content of later targets and marking
// them with Truncated=true. Every target's Content is also sanitized to valid
// UTF-8 so the rendered prompt remains valid even when the underlying source
// (e.g., a git diff carrying binary bytes) is not, and so byte-level truncation
// never lands in the middle of a multi-byte rune. When MaxBytes <= 0, only the
// UTF-8 sanitization runs.
func budget(targets []Target, maxBytes int) []Target {
	out := make([]Target, 0, len(targets))
	if maxBytes <= 0 {
		for _, t := range targets {
			t.Content = strings.ToValidUTF8(t.Content, "")
			out = append(out, t)
		}
		return out
	}
	used := 0
	for _, t := range targets {
		if used+len(t.Content) <= maxBytes {
			used += len(t.Content)
			t.Content = strings.ToValidUTF8(t.Content, "")
			out = append(out, t)
			continue
		}
		remaining := maxBytes - used
		if remaining <= 0 {
			out = append(out, Target{Path: t.Path, Content: "", Truncated: true})
			continue
		}
		truncated := strings.ToValidUTF8(t.Content[:remaining], "")
		out = append(out, Target{Path: t.Path, Content: truncated, Truncated: true})
		used = maxBytes
	}
	return out
}
