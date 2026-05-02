package reviewrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orchv3/internal/agentmeta"
)

func sampleInput(stage agentmeta.Stage) PromptInput {
	return PromptInput{
		Stage:         stage,
		ProducerBy:    "claude",
		ProducerModel: "claude-opus-4-7",
		ReviewerBy:    "codex",
		ReviewerModel: "gpt-5",
		Categories:    []string{"requirement_unclear", "scope_creep", "nit"},
		Targets: []Target{
			{Path: "openspec/changes/foo/proposal.md", Content: "Why: alpha-content"},
			{Path: "openspec/changes/foo/tasks.md", Content: "- [ ] beta-content"},
		},
	}
}

func TestRenderPromptIncludesProducerReviewerStageAndTargets(t *testing.T) {
	in := sampleInput(agentmeta.StageProposal)
	got, err := RenderPrompt(in, "")
	if err != nil {
		t.Fatalf("RenderPrompt: %v", err)
	}

	wantContains := []string{
		"claude",
		"claude-opus-4-7",
		"codex",
		"gpt-5",
		"Стадия: proposal",
		"requirement_unclear",
		"scope_creep",
		"nit",
		"openspec/changes/foo/proposal.md",
		"alpha-content",
		"openspec/changes/foo/tasks.md",
		"beta-content",
	}
	for _, w := range wantContains {
		if !strings.Contains(got, w) {
			t.Errorf("rendered prompt missing %q\n---rendered---\n%s", w, got)
		}
	}
}

func TestRenderPromptForApplyIncludesApplyStageRole(t *testing.T) {
	in := sampleInput(agentmeta.StageApply)
	got, err := RenderPrompt(in, "")
	if err != nil {
		t.Fatalf("RenderPrompt: %v", err)
	}
	for _, w := range []string{"Стадия: apply", "Producer (автор реализации)"} {
		if !strings.Contains(got, w) {
			t.Errorf("rendered apply prompt missing %q", w)
		}
	}
}

func TestRenderPromptForArchiveIncludesArchiveStageRole(t *testing.T) {
	in := sampleInput(agentmeta.StageArchive)
	got, err := RenderPrompt(in, "")
	if err != nil {
		t.Fatalf("RenderPrompt: %v", err)
	}
	for _, w := range []string{"Стадия: archive", "автор архивирования"} {
		if !strings.Contains(got, w) {
			t.Errorf("rendered archive prompt missing %q", w)
		}
	}
}

func TestRenderPromptForUnknownStageFails(t *testing.T) {
	in := sampleInput(agentmeta.Stage("bogus"))
	if _, err := RenderPrompt(in, ""); err == nil {
		t.Fatal("expected error for unknown stage, got nil")
	}
}

func TestRenderPromptUsesOverrideDirWhenFileExists(t *testing.T) {
	dir := t.TempDir()
	sentinel := "SENTINEL-OVERRIDE-XYZ"
	override := "Override template: " + sentinel + "\nProducer={{ .ProducerBy }}\n"
	if err := os.WriteFile(filepath.Join(dir, "proposal_review.tmpl"), []byte(override), 0o644); err != nil {
		t.Fatalf("write override: %v", err)
	}

	in := sampleInput(agentmeta.StageProposal)
	got, err := RenderPrompt(in, dir)
	if err != nil {
		t.Fatalf("RenderPrompt: %v", err)
	}
	if !strings.Contains(got, sentinel) {
		t.Errorf("expected override sentinel %q in output, got: %s", sentinel, got)
	}
	if strings.Contains(got, "OpenSpec-proposal'а") {
		t.Errorf("expected default header NOT to be in output when override is used; got: %s", got)
	}
	if !strings.Contains(got, "claude") {
		t.Errorf("expected override template to be rendered with input; got: %s", got)
	}
}

func TestRenderPromptFallsBackToEmbeddedWhenOverrideDirEmpty(t *testing.T) {
	dir := t.TempDir() // empty - no template file inside
	in := sampleInput(agentmeta.StageProposal)
	got, err := RenderPrompt(in, dir)
	if err != nil {
		t.Fatalf("RenderPrompt: %v", err)
	}
	if !strings.Contains(got, "OpenSpec-proposal'а") {
		t.Errorf("expected embedded default header in output; got: %s", got)
	}
}

func TestRenderPromptMarksTruncatedTargets(t *testing.T) {
	in := sampleInput(agentmeta.StageProposal)
	in.Targets = []Target{
		{Path: "big.md", Content: "trimmed", Truncated: true},
	}
	got, err := RenderPrompt(in, "")
	if err != nil {
		t.Fatalf("RenderPrompt: %v", err)
	}
	if !strings.Contains(got, "big.md (TRUNCATED)") {
		t.Errorf("expected '(TRUNCATED)' marker next to truncated target path; got: %s", got)
	}
}
