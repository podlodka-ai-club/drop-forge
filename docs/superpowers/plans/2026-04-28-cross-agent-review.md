# Cross-Agent Review Stage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Добавить четвёртую стадию оркестрации — Cross-Agent Review. После того как один из существующих producer-runner'ов (`proposalrunner` / `applyrunner` / `archiverunner`) запушил артефакт, оркестратор переводит задачу в новый промежуточный Linear-state. Из него `ReviewRunner` автоматически клонирует ветку, читает producer-trailer последнего коммита, запускает «противоположную» модель как reviewer, парсит strict-JSON ответ и публикует одним POST'ом PR review с inline-комментариями и fix-prompt'ами в стиле CodeRabbit. Решение о слиянии/отклонении остаётся за человеком.

**Architecture:** Новый пакет `internal/reviewrunner` повторяет паттерн существующих stage-runner'ов (temp clone → executor → push). `internal/agentmeta` — общий хелпер producer-trailer для всех runner'ов. `coreorch` получает три новых route'ы (по одной на стадию) и переключает producer-runner'ы на AI-review state через feature-flag. PR review публикуется через `gh api POST /repos/.../pulls/{n}/reviews` с одним атомарным payload'ом и идемпотентным маркером по `(reviewer, stage, HEAD-sha)`. Сегодня оба слота — Codex CLI с разными моделями; Claude позже регистрируется как второй executor без правок в ReviewRunner.

**Tech Stack:** Go 1.22+, `commandrunner` для git/gh, `steplog` для structured logging, `encoding/json` для strict parsing, table-driven тесты с фейковыми executor/PRCommenter/TaskManager.

**Reference:** `docs/superpowers/specs/2026-04-28-cross-agent-review-design.md`

---

## File Structure

**Создаваемые пакеты и файлы:**

- `internal/agentmeta/trailer.go` — `AppendTrailer`, `ParseTrailer`, типы `Producer`, `Stage`.
- `internal/agentmeta/trailer_test.go`
- `internal/reviewrunner/runner.go` — `Runner` struct, `ReviewInput`, `Run()` метод.
- `internal/reviewrunner/runner_test.go`
- `internal/reviewrunner/stage.go` — `Stage` enum (Proposal/Apply/Archive), `StageProfile` со списком категорий и шаблоном targets.
- `internal/reviewrunner/stage_test.go`
- `internal/reviewrunner/targets.go` — функции сбора targets для каждой стадии.
- `internal/reviewrunner/targets_test.go`
- `internal/reviewrunner/prompt.go` — рендеринг prompt-шаблона.
- `internal/reviewrunner/prompt_test.go`
- `internal/reviewrunner/prompts/proposal_review.tmpl`
- `internal/reviewrunner/prompts/apply_review.tmpl`
- `internal/reviewrunner/prompts/archive_review.tmpl`
- `internal/reviewrunner/reviewer.go` — выбор reviewer-slot'а по producer-trailer'у.
- `internal/reviewrunner/reviewer_test.go`
- `internal/reviewrunner/agent_executor.go` — `AgentExecutor` interface (тот же паттерн, что в existing runner'ах).
- `internal/reviewrunner/codex_executor.go` — `CodexCLIExecutor` для review-prompt'а.
- `internal/reviewrunner/reviewparse/parse.go` — типы `Review`, `Summary`, `Finding`, парсер JSON.
- `internal/reviewrunner/reviewparse/parse_test.go`
- `internal/reviewrunner/prcommenter/commenter.go` — interface `PRCommenter`, тип `PostReviewInput`.
- `internal/reviewrunner/prcommenter/gh.go` — `GHPostReviewCommenter` (`gh api`), идемпотентность, форматирование payload'а.
- `internal/reviewrunner/prcommenter/gh_test.go`
- `internal/reviewrunner/prcommenter/format.go` — формирование summary и inline body markdown.
- `internal/reviewrunner/prcommenter/format_test.go`
- `internal/reviewrunner/doc.go` — package doc.

**Модифицируемые файлы:**

- `internal/proposalrunner/runner.go:151-179` — добавить trailer в commit message.
- `internal/applyrunner/runner.go:149-159` — то же.
- `internal/archiverunner/runner.go` — то же (убедиться при чтении файла).
- `internal/coreorch/orchestrator.go:37-47` — расширить `Config` тремя новыми state ID, reviewer-конфигом.
- `internal/coreorch/orchestrator.go:154-211` — feature-flag на target state в `processProposalTask`/`processApplyTask`/`processArchiveTask`.
- `internal/coreorch/orchestrator.go:61-115` — добавить три AI-review route'ы в switch.
- `internal/coreorch/orchestrator.go:362-404` — расширить `validate()`.
- `internal/config/config.go:65-78` — добавить три AI-review state ID; новая структура `ReviewRunnerConfig`.
- `internal/config/config.go:212-258` — `Validate()` и `ManagedStateIDs()` для AI-review states (правило all-or-nothing).
- `internal/taskmanager/...` — без изменений (не трогаем; новый managed state идёт через `LinearTaskManagerConfig.ManagedStateIDs()`).
- `cmd/orchv3/main.go` — wiring `ReviewRunner`, фабрика executor'ов с двумя слотами.
- `architecture.md` — новый раздел «Целевой Поток Review-Stage» + маппинг.
- `docs/proposal-runner.md` — упоминание AI-review этапа.
- `.env.example` — новые переменные.

---

## Task 1: agentmeta package — producer trailer types and helpers

**Files:**
- Create: `internal/agentmeta/trailer.go`
- Create: `internal/agentmeta/trailer_test.go`

Цель: единый хелпер для записи/чтения git-trailer'ов вида `Produced-By:`, `Produced-Model:`, `Produced-Stage:` в commit messages. Это фундамент, на который опираются все три producer-runner'а и `ReviewRunner`.

- [ ] **Step 1: Write failing tests for ParseTrailer**

```go
// internal/agentmeta/trailer_test.go
package agentmeta

import (
	"testing"
)

func TestParseTrailerExtractsAllFields(t *testing.T) {
	message := `Apply: ENG-1: Apply feature

Produced-By: codex
Produced-Model: gpt-5-codex
Produced-Stage: apply
`
	got, err := ParseTrailer(message)
	if err != nil {
		t.Fatalf("ParseTrailer returned error: %v", err)
	}
	want := Producer{By: "codex", Model: "gpt-5-codex", Stage: StageApply}
	if got != want {
		t.Fatalf("ParseTrailer = %+v, want %+v", got, want)
	}
}

func TestParseTrailerIsCaseInsensitiveOnKey(t *testing.T) {
	message := "subject\n\nproduced-by: codex\nPRODUCED-MODEL: gpt-5\nProduced-Stage: proposal\n"
	got, err := ParseTrailer(message)
	if err != nil {
		t.Fatalf("ParseTrailer returned error: %v", err)
	}
	if got.By != "codex" || got.Model != "gpt-5" || got.Stage != StageProposal {
		t.Fatalf("ParseTrailer = %+v", got)
	}
}

func TestParseTrailerReturnsNotFoundWhenAbsent(t *testing.T) {
	message := "subject line only"
	_, err := ParseTrailer(message)
	if err != ErrTrailerNotFound {
		t.Fatalf("err = %v, want ErrTrailerNotFound", err)
	}
}

func TestParseTrailerRejectsUnknownStage(t *testing.T) {
	message := "subject\n\nProduced-By: codex\nProduced-Model: gpt-5\nProduced-Stage: deploy\n"
	_, err := ParseTrailer(message)
	if err == nil {
		t.Fatal("ParseTrailer returned nil error for unknown stage")
	}
}

func TestParseTrailerIgnoresUnrelatedTrailers(t *testing.T) {
	message := `subject

Signed-off-by: Alice <alice@example.com>
Produced-By: codex
Produced-Model: gpt-5-codex
Produced-Stage: archive
Reviewed-by: Bob`
	got, err := ParseTrailer(message)
	if err != nil {
		t.Fatalf("ParseTrailer returned error: %v", err)
	}
	if got.Stage != StageArchive {
		t.Fatalf("Stage = %s, want archive", got.Stage)
	}
}
```

- [ ] **Step 2: Write failing tests for AppendTrailer**

```go
// internal/agentmeta/trailer_test.go (continue)

func TestAppendTrailerAddsAllThreeFields(t *testing.T) {
	got := AppendTrailer("Apply feature", Producer{
		By: "codex", Model: "gpt-5-codex", Stage: StageApply,
	})
	roundTrip, err := ParseTrailer(got)
	if err != nil {
		t.Fatalf("round-trip parse failed: %v\nmessage:\n%s", err, got)
	}
	want := Producer{By: "codex", Model: "gpt-5-codex", Stage: StageApply}
	if roundTrip != want {
		t.Fatalf("round-trip = %+v, want %+v", roundTrip, want)
	}
}

func TestAppendTrailerSeparatesSubjectAndTrailerBlockWithBlankLine(t *testing.T) {
	got := AppendTrailer("Subject line", Producer{By: "codex", Model: "gpt-5", Stage: StageProposal})
	wantLines := []string{
		"Subject line",
		"",
		"Produced-By: codex",
		"Produced-Model: gpt-5",
		"Produced-Stage: proposal",
	}
	gotLines := splitLines(got)
	if !equalStringSlices(gotLines, wantLines) {
		t.Fatalf("AppendTrailer lines = %#v\nwant %#v", gotLines, wantLines)
	}
}

func TestAppendTrailerPreservesExistingTrailerBlock(t *testing.T) {
	original := "Subject\n\nSigned-off-by: Alice <alice@example.com>\n"
	got := AppendTrailer(original, Producer{By: "codex", Model: "gpt-5", Stage: StageProposal})
	roundTrip, err := ParseTrailer(got)
	if err != nil {
		t.Fatalf("round-trip parse failed: %v", err)
	}
	if roundTrip.By != "codex" {
		t.Fatalf("By = %s, want codex", roundTrip.By)
	}
	if !contains(got, "Signed-off-by: Alice") {
		t.Fatalf("AppendTrailer dropped existing trailer block:\n%s", got)
	}
}
```

- [ ] **Step 3: Run tests to verify all fail (no implementation yet)**

Run: `go test ./internal/agentmeta/...`
Expected: build/compile errors — `Producer`, `StageApply`, `ParseTrailer`, `AppendTrailer`, `ErrTrailerNotFound` undefined.

- [ ] **Step 4: Implement trailer.go**

```go
// internal/agentmeta/trailer.go
package agentmeta

import (
	"errors"
	"fmt"
	"strings"
)

type Stage string

const (
	StageProposal Stage = "proposal"
	StageApply    Stage = "apply"
	StageArchive  Stage = "archive"
)

func ParseStage(value string) (Stage, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(StageProposal):
		return StageProposal, nil
	case string(StageApply):
		return StageApply, nil
	case string(StageArchive):
		return StageArchive, nil
	}
	return "", fmt.Errorf("unknown producer stage %q", value)
}

type Producer struct {
	By    string
	Model string
	Stage Stage
}

const (
	keyBy    = "Produced-By"
	keyModel = "Produced-Model"
	keyStage = "Produced-Stage"
)

var ErrTrailerNotFound = errors.New("producer trailer not found")

func AppendTrailer(message string, producer Producer) string {
	message = strings.TrimRight(message, "\n")

	trailer := fmt.Sprintf("%s: %s\n%s: %s\n%s: %s",
		keyBy, producer.By,
		keyModel, producer.Model,
		keyStage, string(producer.Stage),
	)

	if message == "" {
		return trailer + "\n"
	}

	if hasExistingTrailerBlock(message) {
		return message + "\n" + trailer + "\n"
	}

	return message + "\n\n" + trailer + "\n"
}

func ParseTrailer(message string) (Producer, error) {
	lines := strings.Split(message, "\n")

	var byVal, modelVal, stageVal string
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		colonIdx := strings.Index(line, ":")
		if colonIdx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])
		switch strings.ToLower(key) {
		case strings.ToLower(keyBy):
			byVal = value
		case strings.ToLower(keyModel):
			modelVal = value
		case strings.ToLower(keyStage):
			stageVal = value
		}
	}

	if byVal == "" && modelVal == "" && stageVal == "" {
		return Producer{}, ErrTrailerNotFound
	}
	if byVal == "" || modelVal == "" || stageVal == "" {
		return Producer{}, fmt.Errorf("incomplete producer trailer: by=%q model=%q stage=%q", byVal, modelVal, stageVal)
	}

	stage, err := ParseStage(stageVal)
	if err != nil {
		return Producer{}, err
	}
	return Producer{By: byVal, Model: modelVal, Stage: stage}, nil
}

func hasExistingTrailerBlock(message string) bool {
	lines := strings.Split(message, "\n")
	if len(lines) < 3 {
		return false
	}
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			return false
		}
		if !strings.Contains(line, ":") {
			return false
		}
		if i > 0 && strings.TrimSpace(lines[i-1]) == "" {
			return true
		}
	}
	return false
}
```

- [ ] **Step 5: Add test helpers (splitLines, equalStringSlices, contains) at the bottom of trailer_test.go**

```go
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.TrimRight(s, "\n")
	return strings.Split(s, "\n")
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
```

Add `import "strings"` at the top of the test file.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/agentmeta/... -v`
Expected: all 8 tests PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/agentmeta/
git commit -m "Add agentmeta package for producer trailer parsing"
```

---

## Task 2: Wire producer trailer in proposalrunner

**Files:**
- Modify: `internal/proposalrunner/runner.go:151-179`
- Modify: `internal/proposalrunner/runner.go:27-38` (add `Producer` field on Runner)
- Modify: `internal/proposalrunner/runner_test.go` (add assertion on commit message)

Цель: коммиты от `ProposalRunner` несут producer trailer, чтобы `ReviewRunner` мог определить «кто произвёл артефакт». В первой версии producer-данные приходят из конфигурации runner'а — wiring в `cmd/orchv3` появится в Task 16.

- [ ] **Step 1: Read existing runner_test.go to find a happy-path test**

Run: `head -200 internal/proposalrunner/runner_test.go`
Note the test that asserts the `git commit -m "<title>"` command — that's where you'll extend the assertion.

- [ ] **Step 2: Add producer field on Runner struct**

Edit `internal/proposalrunner/runner.go` — add a new field after `Service string`:

```go
type Runner struct {
	Config   config.ProposalRunnerConfig
	Command  commandrunner.Runner
	Agent    AgentExecutor
	Producer agentmeta.Producer  // NEW
	Service  string
	Stdout   io.Writer
	Stderr   io.Writer
	// ... rest unchanged
}
```

Add `"orchv3/internal/agentmeta"` to imports.

When `Producer` is zero-value (no `By`/`Model`/`Stage`), trailer is NOT appended. This keeps existing tests passing if they didn't set Producer.

- [ ] **Step 3: Write failing test asserting trailer in commit message**

Add to `internal/proposalrunner/runner_test.go`:

```go
func TestRunnerCommitMessageContainsProducerTrailerWhenProducerSet(t *testing.T) {
	cfg := validConfig()
	fake := &fakeCommandRunner{
		responses: []fakeResponse{
			{}, // git clone
			{stdout: "{\"url\":\"https://github.com/example/repo/pull/1\"}\n"}, // gh pr create's last call would be later — adjust based on actual order
		},
	}
	// Use the same fake setup as TestRunnerHappyPath... but inject Producer.
	runner := &Runner{
		Config:  cfg,
		Command: fake,
		Agent:   &fakeAgentExecutor{},
		Producer: agentmeta.Producer{
			By: "codex", Model: "gpt-5-codex", Stage: agentmeta.StageProposal,
		},
		Stdout: io.Discard,
		Stderr: io.Discard,
		MkdirTemp: func(dir, pattern string) (string, error) { return t.TempDir(), nil },
		RemoveAll: func(path string) error { return nil },
		Now:       func() time.Time { return time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC) },
	}
	// ... fill rest of fake.responses for the full happy path the same way the existing happy-path test does.
	// Run and find the commit command.

	_, _ = runner.Run(context.Background(), ProposalInput{
		Title: "Test", Identifier: "ZIM-1", AgentPrompt: "ctx",
	})

	commitCmd := findCommandByPrefix(fake.commands, "git", "commit")
	if commitCmd == nil {
		t.Fatal("no git commit command captured")
	}
	commitMessage := commitCmd.Args[len(commitCmd.Args)-1]
	if !strings.Contains(commitMessage, "Produced-By: codex") ||
		!strings.Contains(commitMessage, "Produced-Model: gpt-5-codex") ||
		!strings.Contains(commitMessage, "Produced-Stage: proposal") {
		t.Fatalf("commit message missing trailer:\n%s", commitMessage)
	}
}
```

If `findCommandByPrefix` doesn't exist in this test file, add it:

```go
func findCommandByPrefix(commands []commandrunner.Command, name string, firstArg string) *commandrunner.Command {
	for i := range commands {
		c := &commands[i]
		if c.Name == name && len(c.Args) > 0 && c.Args[0] == firstArg {
			return c
		}
	}
	return nil
}
```

Add imports `"orchv3/internal/agentmeta"`, `"strings"`, `"time"` if missing.

- [ ] **Step 4: Run test to verify it fails**

Run: `go test ./internal/proposalrunner/... -run TestRunnerCommitMessageContainsProducerTrailer -v`
Expected: FAIL — commit message contains only PR title, no trailer.

- [ ] **Step 5: Modify commit message construction**

In `internal/proposalrunner/runner.go:151-179`, replace the `commit -m prTitle` argument:

```go
commitMessage := prTitle
if runner.Producer != (agentmeta.Producer{}) {
	commitMessage = agentmeta.AppendTrailer(prTitle, runner.Producer)
}

gitCommands := []commandrunner.Command{
	{Name: runner.Config.GitPath, Args: []string{"checkout", "-b", branchName}, Dir: cloneDir},
	{Name: runner.Config.GitPath, Args: []string{"add", "-A"}, Dir: cloneDir},
	{Name: runner.Config.GitPath, Args: []string{"commit", "-m", commitMessage}, Dir: cloneDir},
	{Name: runner.Config.GitPath, Args: []string{"push", "-u", runner.Config.RemoteName, branchName}, Dir: cloneDir},
}
```

- [ ] **Step 6: Run tests to verify pass**

Run: `go test ./internal/proposalrunner/...`
Expected: all tests PASS, including the new one and existing ones (which use zero-value Producer).

- [ ] **Step 7: Commit**

```bash
git add internal/proposalrunner/
git commit -m "Wire producer trailer into proposal runner commits"
```

---

## Task 3: Wire producer trailer in applyrunner

**Files:**
- Modify: `internal/applyrunner/runner.go:21-31` (add `Producer` field)
- Modify: `internal/applyrunner/runner.go:149-159` (commit message construction)
- Modify: `internal/applyrunner/runner_test.go` (add new test)

Same pattern as Task 2.

- [ ] **Step 1: Add Producer field to Runner struct**

```go
// internal/applyrunner/runner.go
type Runner struct {
	Config   config.ProposalRunnerConfig
	Command  commandrunner.Runner
	Agent    AgentExecutor
	Producer agentmeta.Producer  // NEW
	Service  string
	// ... rest unchanged
}
```

Add `"orchv3/internal/agentmeta"` import.

- [ ] **Step 2: Write failing test**

Add to `internal/applyrunner/runner_test.go`:

```go
func TestRunnerCommitMessageContainsProducerTrailerWhenProducerSet(t *testing.T) {
	cfg := validConfig()
	fake := &fakeCommandRunner{
		responses: []fakeResponse{
			{},                              // git clone
			{},                              // git checkout
			{stdout: " M file.go\n"},        // git status
			{}, {}, {},                      // add, commit, push
		},
	}
	runner := &Runner{
		Config:  cfg,
		Command: fake,
		Agent:   &fakeAgentExecutor{},
		Producer: agentmeta.Producer{
			By: "codex", Model: "gpt-5-codex", Stage: agentmeta.StageApply,
		},
		Stdout: io.Discard,
		Stderr: io.Discard,
		MkdirTemp: func(dir, pattern string) (string, error) { return t.TempDir(), nil },
		RemoveAll: func(path string) error { return nil },
	}

	err := runner.Run(context.Background(), ApplyInput{
		Title: "Apply feature", Identifier: "ENG-1",
		AgentPrompt: "Task context", BranchName: "feature/task",
	})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	var commitCmd *commandrunner.Command
	for i := range fake.commands {
		c := &fake.commands[i]
		if c.Name == "git" && len(c.Args) > 0 && c.Args[0] == "commit" {
			commitCmd = c
			break
		}
	}
	if commitCmd == nil {
		t.Fatal("no git commit command captured")
	}
	msg := commitCmd.Args[len(commitCmd.Args)-1]
	if !strings.Contains(msg, "Produced-By: codex") ||
		!strings.Contains(msg, "Produced-Stage: apply") {
		t.Fatalf("commit message missing trailer:\n%s", msg)
	}
}
```

Add `"orchv3/internal/agentmeta"`, `"strings"` imports if missing.

- [ ] **Step 3: Run test to verify failure**

Run: `go test ./internal/applyrunner/... -run TestRunnerCommitMessageContainsProducerTrailer -v`
Expected: FAIL.

- [ ] **Step 4: Modify commit message construction**

`internal/applyrunner/runner.go:149-159`:

```go
commitMessage := BuildCommitMessage(input.Identifier, input.Title)
if runner.Producer != (agentmeta.Producer{}) {
	commitMessage = agentmeta.AppendTrailer(commitMessage, runner.Producer)
}

for _, gitCommand := range []commandrunner.Command{
	{Name: runner.Config.GitPath, Args: []string{"add", "-A"}, Dir: cloneDir},
	{Name: runner.Config.GitPath, Args: []string{"commit", "-m", commitMessage}, Dir: cloneDir},
	{Name: runner.Config.GitPath, Args: []string{"push", runner.Config.RemoteName, branchName}, Dir: cloneDir},
} {
	// ... unchanged loop body
}
```

- [ ] **Step 5: Run all applyrunner tests**

Run: `go test ./internal/applyrunner/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/applyrunner/
git commit -m "Wire producer trailer into apply runner commits"
```

---

## Task 4: Wire producer trailer in archiverunner

**Files:**
- Modify: `internal/archiverunner/runner.go` (add `Producer` field, modify commit message)
- Modify: `internal/archiverunner/runner_test.go` (add new test)

Identical pattern to Task 3 but for archiverunner. Read `internal/archiverunner/runner.go` first to find the commit-message construction site.

- [ ] **Step 1: Read archiverunner/runner.go and locate the commit message build**

Run: `grep -n 'commit' internal/archiverunner/runner.go`
Identify where the commit message is built (likely a `BuildCommitMessage` helper or inline).

- [ ] **Step 2: Add `Producer agentmeta.Producer` field and import**

Same pattern as Task 3.

- [ ] **Step 3: Write failing test**

Mirror the test from Task 3 with stage = `agentmeta.StageArchive` and the appropriate fake response sequence for archive flow.

- [ ] **Step 4: Run test to verify failure**

Run: `go test ./internal/archiverunner/... -run TestRunnerCommitMessageContainsProducerTrailer -v`

- [ ] **Step 5: Modify commit message construction**

```go
commitMessage := BuildCommitMessage(input.Identifier, input.Title) // or inline equivalent
if runner.Producer != (agentmeta.Producer{}) {
	commitMessage = agentmeta.AppendTrailer(commitMessage, runner.Producer)
}
```

- [ ] **Step 6: Run all archiverunner tests**

Run: `go test ./internal/archiverunner/...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/archiverunner/
git commit -m "Wire producer trailer into archive runner commits"
```

---

## Task 5: Config — AI-review state IDs and reviewer slot fields

**Files:**
- Modify: `internal/config/config.go:65-78` (extend `LinearTaskManagerConfig`)
- Modify: `internal/config/config.go:35-45` (add `Review ReviewRunnerConfig` to top-level Config)
- Modify: `internal/config/config.go:80-139` (extend `Load()`)
- Modify: `internal/config/config.go:212-258` (extend `Validate()` and `ManagedStateIDs()`)
- Modify: `internal/config/config_test.go`

Цель: пользователь может выставить три AI-review state ID плюс два reviewer-слота с моделью и путём к executor'у. Правило all-or-nothing: либо все три AI-review state ID и оба слота заданы, либо все три AI-review state ID пусты (фича выключена). Частичная конфигурация — ошибка.

- [ ] **Step 1: Write failing tests for config Load and Validate**

Add to `internal/config/config_test.go`:

```go
func TestLoadAIReviewStatesAndReviewerSlots(t *testing.T) {
	t.Setenv("LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID", "p-ai")
	t.Setenv("LINEAR_STATE_NEED_CODE_AI_REVIEW_ID", "c-ai")
	t.Setenv("LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID", "a-ai")
	t.Setenv("REVIEW_ROLE_PRIMARY", "codex")
	t.Setenv("REVIEW_ROLE_SECONDARY", "codex")
	t.Setenv("REVIEW_PRIMARY_MODEL", "gpt-5-codex")
	t.Setenv("REVIEW_SECONDARY_MODEL", "gpt-5")
	t.Setenv("REVIEW_PRIMARY_EXECUTOR_PATH", "/usr/bin/codex")
	t.Setenv("REVIEW_SECONDARY_EXECUTOR_PATH", "/usr/bin/codex")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.TaskManager.NeedProposalAIReviewStateID != "p-ai" {
		t.Fatalf("NeedProposalAIReviewStateID = %s", cfg.TaskManager.NeedProposalAIReviewStateID)
	}
	if cfg.Review.PrimarySlot != "codex" || cfg.Review.SecondarySlot != "codex" {
		t.Fatalf("review slots = %+v", cfg.Review)
	}
	if cfg.Review.PrimaryModel != "gpt-5-codex" || cfg.Review.SecondaryModel != "gpt-5" {
		t.Fatalf("review models = %+v", cfg.Review)
	}
}

func TestLinearTaskManagerConfigValidateAcceptsAllAIReviewEmpty(t *testing.T) {
	cfg := minimalValidLinearConfig()
	// All three AI-review state IDs left empty.
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() returned error for all-empty AI-review states: %v", err)
	}
}

func TestLinearTaskManagerConfigValidateAcceptsAllAIReviewSet(t *testing.T) {
	cfg := minimalValidLinearConfig()
	cfg.NeedProposalAIReviewStateID = "p-ai"
	cfg.NeedCodeAIReviewStateID = "c-ai"
	cfg.NeedArchiveAIReviewStateID = "a-ai"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() returned error for fully-set AI-review states: %v", err)
	}
}

func TestLinearTaskManagerConfigValidateRejectsPartialAIReview(t *testing.T) {
	cfg := minimalValidLinearConfig()
	cfg.NeedProposalAIReviewStateID = "p-ai"
	// Code and archive AI-review left empty.
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() accepted partial AI-review configuration")
	}
	if !strings.Contains(err.Error(), "AI review") {
		t.Fatalf("Validate() error = %v, want mention of AI review", err)
	}
}

func TestManagedStateIDsIncludesAIReviewStatesWhenSet(t *testing.T) {
	cfg := LinearTaskManagerConfig{
		ReadyToProposeStateID:        "p",
		ReadyToCodeStateID:           "c",
		ReadyToArchiveStateID:        "a",
		NeedProposalAIReviewStateID:  "p-ai",
		NeedCodeAIReviewStateID:      "c-ai",
		NeedArchiveAIReviewStateID:   "a-ai",
	}
	got := cfg.ManagedStateIDs()
	want := []string{"p", "c", "a", "p-ai", "c-ai", "a-ai"}
	if !equalStringSlicesUnordered(got, want) {
		t.Fatalf("ManagedStateIDs = %v, want %v", got, want)
	}
}
```

If `minimalValidLinearConfig()` and `equalStringSlicesUnordered()` helpers don't exist, add them in this test file:

```go
func minimalValidLinearConfig() LinearTaskManagerConfig {
	return LinearTaskManagerConfig{
		APIURL:                     defaultLinearAPIURL,
		APIToken:                   "tok",
		ProjectID:                  "prj",
		ReadyToProposeStateID:      "p",
		ReadyToCodeStateID:         "c",
		ReadyToArchiveStateID:      "a",
		ProposingInProgressStateID: "pip",
		CodeInProgressStateID:      "cip",
		ArchivingInProgressStateID: "aip",
		NeedProposalReviewStateID:  "npr",
		NeedCodeReviewStateID:      "ncr",
		NeedArchiveReviewStateID:   "nar",
	}
}

func equalStringSlicesUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[string]int{}
	for _, x := range a {
		m[x]++
	}
	for _, x := range b {
		m[x]--
		if m[x] < 0 {
			return false
		}
	}
	return true
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/config/... -run "AIReview|ManagedStateIDsIncludesAI" -v`
Expected: compile errors — fields and `ReviewRunnerConfig` undefined.

- [ ] **Step 3: Add new fields to `LinearTaskManagerConfig`**

```go
// internal/config/config.go
type LinearTaskManagerConfig struct {
	APIURL                       string
	APIToken                     string
	ProjectID                    string
	ReadyToProposeStateID        string
	ReadyToCodeStateID           string
	ReadyToArchiveStateID        string
	ProposingInProgressStateID   string
	CodeInProgressStateID        string
	ArchivingInProgressStateID   string
	NeedProposalReviewStateID    string
	NeedCodeReviewStateID        string
	NeedArchiveReviewStateID     string
	NeedProposalAIReviewStateID  string // NEW
	NeedCodeAIReviewStateID      string // NEW
	NeedArchiveAIReviewStateID   string // NEW
}
```

- [ ] **Step 4: Add `ReviewRunnerConfig` and Config field**

```go
// internal/config/config.go (after LinearTaskManagerConfig)

type ReviewRunnerConfig struct {
	PrimarySlot          string
	SecondarySlot        string
	PrimaryModel         string
	SecondaryModel       string
	PrimaryExecutorPath  string
	SecondaryExecutorPath string
	MaxContextBytes      int
	ParseRepairRetries   int
	PromptDir            string
}

const (
	defaultReviewMaxContextBytes    = 256 * 1024
	defaultReviewParseRepairRetries = 1
)
```

```go
type Config struct {
	AppEnv               string
	AppName              string
	LogLevel             string
	HTTPPort             int
	OpenAIAPIKey         string
	ProposalPollInterval time.Duration
	ProposalRunner       ProposalRunnerConfig
	TaskManager          LinearTaskManagerConfig
	Review               ReviewRunnerConfig // NEW
	Logstash             LogstashConfig
}
```

- [ ] **Step 5: Extend Load() to read new env vars**

Inside `Load()`, after the existing `TaskManager:` block, add the three new state IDs. Then add a Review block:

```go
// In Load(), inside the returned struct literal:
TaskManager: LinearTaskManagerConfig{
	// ... existing fields ...
	NeedProposalAIReviewStateID:  trimmedStringFromEnv("LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID", ""),
	NeedCodeAIReviewStateID:      trimmedStringFromEnv("LINEAR_STATE_NEED_CODE_AI_REVIEW_ID", ""),
	NeedArchiveAIReviewStateID:   trimmedStringFromEnv("LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID", ""),
},
Review: ReviewRunnerConfig{
	PrimarySlot:           trimmedStringFromEnv("REVIEW_ROLE_PRIMARY", ""),
	SecondarySlot:         trimmedStringFromEnv("REVIEW_ROLE_SECONDARY", ""),
	PrimaryModel:          trimmedStringFromEnv("REVIEW_PRIMARY_MODEL", ""),
	SecondaryModel:        trimmedStringFromEnv("REVIEW_SECONDARY_MODEL", ""),
	PrimaryExecutorPath:   trimmedStringFromEnv("REVIEW_PRIMARY_EXECUTOR_PATH", ""),
	SecondaryExecutorPath: trimmedStringFromEnv("REVIEW_SECONDARY_EXECUTOR_PATH", ""),
	MaxContextBytes:       reviewMaxBytes,
	ParseRepairRetries:    reviewRetries,
	PromptDir:             trimmedStringFromEnv("REVIEW_PROMPT_DIR", ""),
},
```

Above the struct literal, parse the int env vars:

```go
reviewMaxBytes, err := intFromEnv("REVIEW_MAX_CONTEXT_BYTES", defaultReviewMaxContextBytes)
if err != nil {
	return Config{}, err
}
reviewRetries, err := intFromEnv("REVIEW_PARSE_REPAIR_RETRIES", defaultReviewParseRepairRetries)
if err != nil {
	return Config{}, err
}
```

- [ ] **Step 6: Extend Validate() with all-or-nothing rule**

```go
// internal/config/config.go inside (cfg LinearTaskManagerConfig) Validate()
func (cfg LinearTaskManagerConfig) Validate() error {
	requiredValues := map[string]string{
		// ... existing entries unchanged ...
	}
	for key, value := range requiredValues {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must not be empty", key)
		}
	}

	aiStates := map[string]string{
		"LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID": cfg.NeedProposalAIReviewStateID,
		"LINEAR_STATE_NEED_CODE_AI_REVIEW_ID":     cfg.NeedCodeAIReviewStateID,
		"LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID":  cfg.NeedArchiveAIReviewStateID,
	}
	emptyAI := 0
	for _, v := range aiStates {
		if strings.TrimSpace(v) == "" {
			emptyAI++
		}
	}
	if emptyAI != 0 && emptyAI != len(aiStates) {
		var missing []string
		for k, v := range aiStates {
			if strings.TrimSpace(v) == "" {
				missing = append(missing, k)
			}
		}
		return fmt.Errorf("AI review configuration is partial; set all three or none. Missing: %s", strings.Join(missing, ", "))
	}

	return nil
}
```

- [ ] **Step 7: Extend ManagedStateIDs**

```go
func (cfg LinearTaskManagerConfig) ManagedStateIDs() []string {
	states := []string{
		strings.TrimSpace(cfg.ReadyToProposeStateID),
		strings.TrimSpace(cfg.ReadyToCodeStateID),
		strings.TrimSpace(cfg.ReadyToArchiveStateID),
		strings.TrimSpace(cfg.NeedProposalAIReviewStateID),
		strings.TrimSpace(cfg.NeedCodeAIReviewStateID),
		strings.TrimSpace(cfg.NeedArchiveAIReviewStateID),
	}

	result := make([]string, 0, len(states))
	seen := make(map[string]struct{}, len(states))
	for _, state := range states {
		if state == "" {
			continue
		}
		if _, ok := seen[state]; ok {
			continue
		}
		seen[state] = struct{}{}
		result = append(result, state)
	}
	return result
}
```

- [ ] **Step 8: Add helper on `ReviewRunnerConfig` for feature-flag check**

```go
// AIReviewEnabled reports whether the cross-agent review stage is configured to run.
// True iff all three AI-review state IDs are set on the linear task manager config
// AND both reviewer slots are configured. The caller passes the linear config in.
func (cfg ReviewRunnerConfig) Enabled(linear LinearTaskManagerConfig) bool {
	if strings.TrimSpace(linear.NeedProposalAIReviewStateID) == "" ||
		strings.TrimSpace(linear.NeedCodeAIReviewStateID) == "" ||
		strings.TrimSpace(linear.NeedArchiveAIReviewStateID) == "" {
		return false
	}
	if strings.TrimSpace(cfg.PrimarySlot) == "" || strings.TrimSpace(cfg.SecondarySlot) == "" {
		return false
	}
	if strings.TrimSpace(cfg.PrimaryModel) == "" || strings.TrimSpace(cfg.SecondaryModel) == "" {
		return false
	}
	if strings.TrimSpace(cfg.PrimaryExecutorPath) == "" || strings.TrimSpace(cfg.SecondaryExecutorPath) == "" {
		return false
	}
	return true
}
```

- [ ] **Step 9: Run all config tests**

Run: `go test ./internal/config/...`
Expected: PASS.

- [ ] **Step 10: Commit**

```bash
git add internal/config/
git commit -m "Add AI review state IDs and reviewer slot configuration"
```

---

## Task 6: TaskManager — verify ManagedStateIDs flows through

**Files:**
- Modify: `internal/taskmanager/taskmanager_test.go`

`TaskManager.GetTasks` already uses `cfg.ManagedStateIDs()` (see `taskmanager.go:76`). Task 5 already extended ManagedStateIDs. This task is a regression-safety check: assert that AI-review state IDs end up in the Linear query.

- [ ] **Step 1: Find existing test that verifies state IDs passed to Linear client**

Run: `grep -n "stateIDs" internal/taskmanager/taskmanager_test.go`
Locate the test where a fake client captures the `stateIDs` argument.

- [ ] **Step 2: Add a test asserting AI-review state IDs are included**

Add a new test next to the existing one:

```go
func TestGetTasksIncludesAIReviewStateIDsWhenConfigured(t *testing.T) {
	cfg := config.LinearTaskManagerConfig{
		APIURL: "https://x", APIToken: "t", ProjectID: "p",
		ReadyToProposeStateID:        "r1",
		ReadyToCodeStateID:           "r2",
		ReadyToArchiveStateID:        "r3",
		ProposingInProgressStateID:   "pip",
		CodeInProgressStateID:        "cip",
		ArchivingInProgressStateID:   "aip",
		NeedProposalReviewStateID:    "npr",
		NeedCodeReviewStateID:        "ncr",
		NeedArchiveReviewStateID:     "nar",
		NeedProposalAIReviewStateID:  "npai",
		NeedCodeAIReviewStateID:      "ncai",
		NeedArchiveAIReviewStateID:   "naai",
	}
	captured := &recordingClient{}
	mgr := &Manager{Config: cfg, Client: captured, LogWriter: io.Discard}
	if _, err := mgr.GetTasks(context.Background()); err != nil {
		t.Fatalf("GetTasks: %v", err)
	}
	wantSubset := []string{"npai", "ncai", "naai"}
	for _, want := range wantSubset {
		if !contains(captured.lastStateIDs, want) {
			t.Fatalf("state IDs %v missing %s", captured.lastStateIDs, want)
		}
	}
}

func contains(s []string, v string) bool {
	for _, x := range s { if x == v { return true } }
	return false
}
```

If `recordingClient` is named differently, use that. If the existing test file uses `fakeClient` with a stateIDs capture, mirror that.

- [ ] **Step 3: Run the test**

Run: `go test ./internal/taskmanager/... -run TestGetTasksIncludesAIReviewStateIDsWhenConfigured -v`
Expected: PASS (no implementation changes needed — this verifies Task 5 wired correctly).

- [ ] **Step 4: Commit**

```bash
git add internal/taskmanager/
git commit -m "Cover AI review state IDs in task manager tests"
```

---

## Task 7: reviewparse package — JSON schema and parser

**Files:**
- Create: `internal/reviewrunner/reviewparse/parse.go`
- Create: `internal/reviewrunner/reviewparse/parse_test.go`

Цель: типы для review-ответа модели и строгий парсер с детальными ошибками. Парсер используется в `ReviewRunner` для разбора первого ответа executor'а и repair-ответа.

- [ ] **Step 1: Write failing tests for parse.go**

```go
// internal/reviewrunner/reviewparse/parse_test.go
package reviewparse

import (
	"testing"

	"orchv3/internal/agentmeta"
)

func TestParseValidProposalReview(t *testing.T) {
	raw := `{
		"summary": {
			"verdict": "needs-work",
			"walkthrough": "Proposal looks ok",
			"stats": {"findings": 1, "by_severity": {"blocker":0,"major":1,"minor":0,"nit":0}}
		},
		"findings": [
			{"id":"F1","category":"requirement_unclear","severity":"major",
			 "file":"openspec/changes/x/proposal.md","line_start":10,"line_end":14,
			 "title":"Unclear","message":"...","fix_prompt":"..."}
		]
	}`
	got, err := Parse([]byte(raw), agentmeta.StageProposal)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if got.Summary.Verdict != VerdictNeedsWork {
		t.Fatalf("verdict = %s", got.Summary.Verdict)
	}
	if len(got.Findings) != 1 || got.Findings[0].Category != "requirement_unclear" {
		t.Fatalf("findings = %+v", got.Findings)
	}
}

func TestParseRejectsUnknownVerdict(t *testing.T) {
	raw := `{"summary":{"verdict":"awesome","walkthrough":"x","stats":{"findings":0,"by_severity":{"blocker":0,"major":0,"minor":0,"nit":0}}},"findings":[]}`
	_, err := Parse([]byte(raw), agentmeta.StageProposal)
	if err == nil || !contains(err.Error(), "verdict") {
		t.Fatalf("expected verdict error, got %v", err)
	}
}

func TestParseRejectsUnknownSeverity(t *testing.T) {
	raw := `{"summary":{"verdict":"ship-ready","walkthrough":"x","stats":{"findings":1,"by_severity":{"blocker":0,"major":0,"minor":0,"nit":0}}},"findings":[{"id":"F1","category":"nit","severity":"critical","file":"a","line_start":null,"line_end":null,"title":"t","message":"m","fix_prompt":"p"}]}`
	_, err := Parse([]byte(raw), agentmeta.StageProposal)
	if err == nil || !contains(err.Error(), "severity") {
		t.Fatalf("expected severity error, got %v", err)
	}
}

func TestParseRejectsCategoryNotInStageEnum(t *testing.T) {
	// 'bug' is an Apply-stage category; passing in Proposal-stage must fail.
	raw := `{"summary":{"verdict":"needs-work","walkthrough":"x","stats":{"findings":1,"by_severity":{"blocker":0,"major":1,"minor":0,"nit":0}}},"findings":[{"id":"F1","category":"bug","severity":"major","file":"a","line_start":1,"line_end":1,"title":"t","message":"m","fix_prompt":"p"}]}`
	_, err := Parse([]byte(raw), agentmeta.StageProposal)
	if err == nil || !contains(err.Error(), "category") {
		t.Fatalf("expected category error, got %v", err)
	}
}

func TestParseAllowsNullLineRangeForGeneralFindings(t *testing.T) {
	raw := `{"summary":{"verdict":"ship-ready","walkthrough":"ok","stats":{"findings":1,"by_severity":{"blocker":0,"major":0,"minor":0,"nit":1}}},"findings":[{"id":"F1","category":"nit","severity":"nit","file":"openspec/x.md","line_start":null,"line_end":null,"title":"t","message":"m","fix_prompt":"p"}]}`
	got, err := Parse([]byte(raw), agentmeta.StageProposal)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if got.Findings[0].LineStart != nil || got.Findings[0].LineEnd != nil {
		t.Fatalf("expected null line range, got %+v", got.Findings[0])
	}
}

func TestParseRejectsMalformedJSON(t *testing.T) {
	_, err := Parse([]byte("not-json"), agentmeta.StageProposal)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func contains(s, sub string) bool { return len(s) >= len(sub) && (s == sub || (len(s) > 0 && len(sub) > 0 && stringIndex(s, sub) >= 0)) }
func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub { return i }
	}
	return -1
}
```

- [ ] **Step 2: Implement parse.go**

```go
// internal/reviewrunner/reviewparse/parse.go
package reviewparse

import (
	"encoding/json"
	"fmt"

	"orchv3/internal/agentmeta"
)

type Verdict string

const (
	VerdictShipReady Verdict = "ship-ready"
	VerdictNeedsWork Verdict = "needs-work"
	VerdictBlocked   Verdict = "blocked"
)

type Severity string

const (
	SeverityBlocker Severity = "blocker"
	SeverityMajor   Severity = "major"
	SeverityMinor   Severity = "minor"
	SeverityNit     Severity = "nit"
)

type Stats struct {
	Findings   int            `json:"findings"`
	BySeverity map[string]int `json:"by_severity"`
}

type Summary struct {
	Verdict     Verdict `json:"verdict"`
	Walkthrough string  `json:"walkthrough"`
	Stats       Stats   `json:"stats"`
}

type Finding struct {
	ID        string   `json:"id"`
	Category  string   `json:"category"`
	Severity  Severity `json:"severity"`
	File      string   `json:"file"`
	LineStart *int     `json:"line_start"`
	LineEnd   *int     `json:"line_end"`
	Title     string   `json:"title"`
	Message   string   `json:"message"`
	FixPrompt string   `json:"fix_prompt"`
}

type Review struct {
	Summary  Summary   `json:"summary"`
	Findings []Finding `json:"findings"`
}

var validVerdicts = map[Verdict]struct{}{
	VerdictShipReady: {},
	VerdictNeedsWork: {},
	VerdictBlocked:   {},
}

var validSeverities = map[Severity]struct{}{
	SeverityBlocker: {},
	SeverityMajor:   {},
	SeverityMinor:   {},
	SeverityNit:     {},
}

func Parse(raw []byte, stage agentmeta.Stage) (Review, error) {
	var r Review
	if err := json.Unmarshal(raw, &r); err != nil {
		return Review{}, fmt.Errorf("decode review JSON: %w", err)
	}
	if _, ok := validVerdicts[r.Summary.Verdict]; !ok {
		return Review{}, fmt.Errorf("invalid summary.verdict %q", r.Summary.Verdict)
	}

	allowedCategories, err := CategoriesForStage(stage)
	if err != nil {
		return Review{}, err
	}

	for i, f := range r.Findings {
		if _, ok := validSeverities[f.Severity]; !ok {
			return Review{}, fmt.Errorf("findings[%d].severity %q is not in {blocker, major, minor, nit}", i, f.Severity)
		}
		if _, ok := allowedCategories[f.Category]; !ok {
			return Review{}, fmt.Errorf("findings[%d].category %q not allowed for stage %s", i, f.Category, stage)
		}
		if (f.LineStart == nil) != (f.LineEnd == nil) {
			return Review{}, fmt.Errorf("findings[%d]: line_start and line_end must both be set or both null", i)
		}
	}
	return r, nil
}

// CategoriesForStage is implemented in stage_categories.go (Task 8 will create).
// For now Task 7 stubs it inline and Task 8 replaces with real per-stage enum.
func CategoriesForStage(stage agentmeta.Stage) (map[string]struct{}, error) {
	return nil, fmt.Errorf("stage categories not yet wired; complete Task 8")
}
```

This intentionally fails the category-check test until Task 8 — but the parser shape is finalized. Adjust Task 7 tests to skip category-check tests until Task 8 lands, OR write Task 7 + Task 8 atomically. **Choice: write Tasks 7 and 8 together to avoid temporary broken state.** Skip the commit in Step 5 below until Task 8 is done.

- [ ] **Step 3: Stop here, proceed to Task 8 in the same session**

Do not commit yet. Continue to Task 8.

---

## Task 8: Stage profiles — categories, verdict computation, severity icons

**Files:**
- Create: `internal/reviewrunner/stage.go`
- Create: `internal/reviewrunner/stage_test.go`
- Create: `internal/reviewrunner/reviewparse/categories.go`
- Modify: `internal/reviewrunner/reviewparse/parse.go` (replace stub `CategoriesForStage`)

- [ ] **Step 1: Write failing tests**

```go
// internal/reviewrunner/stage_test.go
package reviewrunner

import (
	"testing"

	"orchv3/internal/agentmeta"
	"orchv3/internal/reviewrunner/reviewparse"
)

func TestStageProfileForProposalHasExpectedCategories(t *testing.T) {
	got := MustProfile(agentmeta.StageProposal)
	for _, want := range []string{"requirement_unclear", "scenario_missing", "scope_creep", "tasks_misaligned", "architecture_violation", "nit"} {
		if _, ok := got.Categories[want]; !ok {
			t.Fatalf("Proposal categories missing %s", want)
		}
	}
}

func TestStageProfileForApplyHasExpectedCategories(t *testing.T) {
	got := MustProfile(agentmeta.StageApply)
	for _, want := range []string{"spec_mismatch", "bug", "concurrency", "test_gap", "config_drift", "idiom"} {
		if _, ok := got.Categories[want]; !ok {
			t.Fatalf("Apply categories missing %s", want)
		}
	}
}

func TestStageProfileForArchiveHasExpectedCategories(t *testing.T) {
	got := MustProfile(agentmeta.StageArchive)
	for _, want := range []string{"incomplete_archive", "spec_drift", "dangling_reference", "metadata_missing", "nit"} {
		if _, ok := got.Categories[want]; !ok {
			t.Fatalf("Archive categories missing %s", want)
		}
	}
}

func TestComputeVerdictBlockedWhenAnyBlocker(t *testing.T) {
	got := ComputeVerdict([]reviewparse.Finding{{Severity: reviewparse.SeverityNit}, {Severity: reviewparse.SeverityBlocker}})
	if got != reviewparse.VerdictBlocked {
		t.Fatalf("verdict = %s", got)
	}
}

func TestComputeVerdictNeedsWorkWhenMajorButNoBlocker(t *testing.T) {
	got := ComputeVerdict([]reviewparse.Finding{{Severity: reviewparse.SeverityMajor}, {Severity: reviewparse.SeverityNit}})
	if got != reviewparse.VerdictNeedsWork {
		t.Fatalf("verdict = %s", got)
	}
}

func TestComputeVerdictShipReadyWhenOnlyMinorAndNit(t *testing.T) {
	got := ComputeVerdict([]reviewparse.Finding{{Severity: reviewparse.SeverityMinor}, {Severity: reviewparse.SeverityNit}})
	if got != reviewparse.VerdictShipReady {
		t.Fatalf("verdict = %s", got)
	}
}

func TestSeverityIconForEachLevel(t *testing.T) {
	cases := map[reviewparse.Severity]string{
		reviewparse.SeverityBlocker: "🛑",
		reviewparse.SeverityMajor:   "⚠️",
		reviewparse.SeverityMinor:   "💡",
		reviewparse.SeverityNit:     "🪶",
	}
	for sev, want := range cases {
		if got := SeverityIcon(sev); got != want {
			t.Fatalf("SeverityIcon(%s) = %s, want %s", sev, got, want)
		}
	}
}
```

- [ ] **Step 2: Implement stage.go**

```go
// internal/reviewrunner/stage.go
package reviewrunner

import (
	"fmt"

	"orchv3/internal/agentmeta"
	"orchv3/internal/reviewrunner/reviewparse"
)

type StageProfile struct {
	Stage      agentmeta.Stage
	Categories map[string]struct{}
	PromptName string
}

var profiles = map[agentmeta.Stage]StageProfile{
	agentmeta.StageProposal: {
		Stage:      agentmeta.StageProposal,
		PromptName: "proposal_review.tmpl",
		Categories: setOf(
			"requirement_unclear",
			"requirement_contradicts_existing",
			"scenario_missing",
			"acceptance_criteria_weak",
			"scope_creep",
			"tasks_misaligned",
			"architecture_violation",
			"nit",
		),
	},
	agentmeta.StageApply: {
		Stage:      agentmeta.StageApply,
		PromptName: "apply_review.tmpl",
		Categories: setOf(
			"spec_mismatch",
			"bug",
			"error_handling",
			"concurrency",
			"test_gap",
			"architecture_violation",
			"idiom",
			"config_drift",
			"nit",
		),
	},
	agentmeta.StageArchive: {
		Stage:      agentmeta.StageArchive,
		PromptName: "archive_review.tmpl",
		Categories: setOf(
			"incomplete_archive",
			"spec_drift",
			"dangling_reference",
			"metadata_missing",
			"nit",
		),
	},
}

func ProfileFor(stage agentmeta.Stage) (StageProfile, error) {
	p, ok := profiles[stage]
	if !ok {
		return StageProfile{}, fmt.Errorf("no stage profile for %s", stage)
	}
	return p, nil
}

func MustProfile(stage agentmeta.Stage) StageProfile {
	p, err := ProfileFor(stage)
	if err != nil {
		panic(err)
	}
	return p
}

func ComputeVerdict(findings []reviewparse.Finding) reviewparse.Verdict {
	hasBlocker, hasMajor := false, false
	for _, f := range findings {
		switch f.Severity {
		case reviewparse.SeverityBlocker:
			hasBlocker = true
		case reviewparse.SeverityMajor:
			hasMajor = true
		}
	}
	switch {
	case hasBlocker:
		return reviewparse.VerdictBlocked
	case hasMajor:
		return reviewparse.VerdictNeedsWork
	default:
		return reviewparse.VerdictShipReady
	}
}

func SeverityIcon(s reviewparse.Severity) string {
	switch s {
	case reviewparse.SeverityBlocker:
		return "🛑"
	case reviewparse.SeverityMajor:
		return "⚠️"
	case reviewparse.SeverityMinor:
		return "💡"
	case reviewparse.SeverityNit:
		return "🪶"
	}
	return "•"
}

func setOf(values ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(values))
	for _, v := range values {
		m[v] = struct{}{}
	}
	return m
}
```

- [ ] **Step 3: Replace stub in reviewparse**

Create `internal/reviewrunner/reviewparse/categories.go`:

```go
package reviewparse

import (
	"fmt"

	"orchv3/internal/agentmeta"
)

var stageCategories = map[agentmeta.Stage]map[string]struct{}{
	agentmeta.StageProposal: setOf(
		"requirement_unclear",
		"requirement_contradicts_existing",
		"scenario_missing",
		"acceptance_criteria_weak",
		"scope_creep",
		"tasks_misaligned",
		"architecture_violation",
		"nit",
	),
	agentmeta.StageApply: setOf(
		"spec_mismatch",
		"bug",
		"error_handling",
		"concurrency",
		"test_gap",
		"architecture_violation",
		"idiom",
		"config_drift",
		"nit",
	),
	agentmeta.StageArchive: setOf(
		"incomplete_archive",
		"spec_drift",
		"dangling_reference",
		"metadata_missing",
		"nit",
	),
}

func setOf(values ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(values))
	for _, v := range values {
		m[v] = struct{}{}
	}
	return m
}
```

Replace the stub in `parse.go`:

```go
func CategoriesForStage(stage agentmeta.Stage) (map[string]struct{}, error) {
	cats, ok := stageCategories[stage]
	if !ok {
		return nil, fmt.Errorf("no categories registered for stage %s", stage)
	}
	return cats, nil
}
```

(Remove the temporary stub error.)

- [ ] **Step 4: Run all reviewrunner and reviewparse tests**

Run: `go test ./internal/reviewrunner/...`
Expected: PASS for stage_test.go AND for reviewparse/parse_test.go.

- [ ] **Step 5: Commit Tasks 7 + 8 together**

```bash
git add internal/reviewrunner/stage.go internal/reviewrunner/stage_test.go internal/reviewrunner/reviewparse/
git commit -m "Add reviewparse and stage profile primitives"
```

---

## Task 9: Targets collection

**Files:**
- Create: `internal/reviewrunner/targets.go`
- Create: `internal/reviewrunner/targets_test.go`

Цель: для каждой стадии собрать пакет файлов и diff'ов из cloned repo, который пойдёт в prompt. Контракт: возвращает `[]Target`, где каждый Target — `{Path string, Content string, Truncated bool}`. При превышении `MaxBytes` усекает по приоритетам.

- [ ] **Step 1: Write failing tests**

```go
// internal/reviewrunner/targets_test.go
package reviewrunner

import (
	"os"
	"path/filepath"
	"testing"

	"orchv3/internal/agentmeta"
)

func TestCollectTargetsForProposalReadsAllChangeFiles(t *testing.T) {
	repo := t.TempDir()
	change := filepath.Join(repo, "openspec", "changes", "x")
	if err := os.MkdirAll(filepath.Join(change, "specs", "cap"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(change, "proposal.md"), "## Why\nbecause")
	writeFile(t, filepath.Join(change, "design.md"), "design body")
	writeFile(t, filepath.Join(change, "tasks.md"), "tasks body")
	writeFile(t, filepath.Join(change, "specs", "cap", "spec.md"), "spec body")

	got, err := CollectTargets(TargetInput{
		Stage:    agentmeta.StageProposal,
		CloneDir: repo,
		MaxBytes: 1 << 20,
		ChangePath: filepath.Join("openspec", "changes", "x"),
	})
	if err != nil {
		t.Fatalf("CollectTargets: %v", err)
	}
	wantPaths := []string{
		"openspec/changes/x/proposal.md",
		"openspec/changes/x/design.md",
		"openspec/changes/x/tasks.md",
		"openspec/changes/x/specs/cap/spec.md",
	}
	if !targetsHavePaths(got, wantPaths) {
		t.Fatalf("paths %v missing in %v", wantPaths, targetPaths(got))
	}
}

func TestCollectTargetsTruncatesAndMarksWhenOverBudget(t *testing.T) {
	repo := t.TempDir()
	change := filepath.Join(repo, "openspec", "changes", "x")
	if err := os.MkdirAll(change, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(change, "proposal.md"), repeat("a", 1024))
	writeFile(t, filepath.Join(change, "design.md"), repeat("b", 4096))

	got, err := CollectTargets(TargetInput{
		Stage:      agentmeta.StageProposal,
		CloneDir:   repo,
		MaxBytes:   2048,
		ChangePath: filepath.Join("openspec", "changes", "x"),
	})
	if err != nil {
		t.Fatalf("CollectTargets: %v", err)
	}
	totalBytes := 0
	truncated := false
	for _, target := range got {
		totalBytes += len(target.Content)
		if target.Truncated {
			truncated = true
		}
	}
	if totalBytes > 2048 {
		t.Fatalf("totalBytes=%d > MaxBytes=2048", totalBytes)
	}
	if !truncated {
		t.Fatal("expected at least one Truncated target")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { t.Fatal(err) }
}
func repeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ { out = append(out, s...) }
	return string(out)
}
func targetPaths(ts []Target) []string {
	out := make([]string, len(ts))
	for i, t := range ts { out[i] = t.Path }
	return out
}
func targetsHavePaths(got []Target, want []string) bool {
	have := map[string]bool{}
	for _, t := range got { have[t.Path] = true }
	for _, w := range want { if !have[w] { return false } }
	return true
}
```

- [ ] **Step 2: Implement targets.go**

```go
// internal/reviewrunner/targets.go
package reviewrunner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"orchv3/internal/agentmeta"
)

type Target struct {
	Path      string
	Content   string
	Truncated bool
}

type TargetInput struct {
	Stage      agentmeta.Stage
	CloneDir   string
	MaxBytes   int
	ChangePath string // relative path inside clone dir of the OpenSpec change being reviewed
	Diff       string // git diff output for Apply/Archive stages
}

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
	root := filepath.Join(in.CloneDir, in.ChangePath)
	files, err := walkMarkdownFiles(root)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)

	out := make([]Target, 0, len(files))
	for _, abs := range files {
		rel, _ := filepath.Rel(in.CloneDir, abs)
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
	out := []Target{
		{Path: "<diff>", Content: in.Diff},
	}
	if in.ChangePath != "" {
		props, err := collectProposal(TargetInput{
			Stage:      agentmeta.StageProposal,
			CloneDir:   in.CloneDir,
			ChangePath: in.ChangePath,
			MaxBytes:   in.MaxBytes,
		})
		if err == nil {
			out = append(out, props...)
		}
	}
	return budget(out, in.MaxBytes), nil
}

func collectArchive(in TargetInput) ([]Target, error) {
	out := []Target{{Path: "<diff>", Content: in.Diff}}
	return budget(out, in.MaxBytes), nil
}

func walkMarkdownFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil { return err }
		if info.IsDir() { return nil }
		if strings.HasSuffix(path, ".md") { files = append(files, path) }
		return nil
	})
	return files, err
}

func budget(targets []Target, maxBytes int) []Target {
	if maxBytes <= 0 { return targets }
	used := 0
	out := make([]Target, 0, len(targets))
	for _, t := range targets {
		if used+len(t.Content) <= maxBytes {
			used += len(t.Content)
			out = append(out, t)
			continue
		}
		remaining := maxBytes - used
		if remaining <= 0 {
			out = append(out, Target{Path: t.Path, Content: "", Truncated: true})
			continue
		}
		out = append(out, Target{Path: t.Path, Content: t.Content[:remaining], Truncated: true})
		used = maxBytes
	}
	return out
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/reviewrunner/... -run Targets -v`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/reviewrunner/targets.go internal/reviewrunner/targets_test.go
git commit -m "Add target collection for review runner stages"
```

---

## Task 10: Prompt templates and renderer

**Files:**
- Create: `internal/reviewrunner/prompts/proposal_review.tmpl`
- Create: `internal/reviewrunner/prompts/apply_review.tmpl`
- Create: `internal/reviewrunner/prompts/archive_review.tmpl`
- Create: `internal/reviewrunner/prompt.go`
- Create: `internal/reviewrunner/prompt_test.go`

Цель: prompt-шаблоны на русском с английскими keyword'ами, plus renderer, который встраивает producer-инфо, JSON-схему, список категорий и pack of targets.

- [ ] **Step 1: Write the proposal_review.tmpl**

Save to `internal/reviewrunner/prompts/proposal_review.tmpl`:

```
Ты — рецензент OpenSpec-проposal'а.

Producer (автор спецификации): {{ .ProducerBy }} / модель {{ .ProducerModel }}.
Reviewer (ты): {{ .ReviewerBy }} / модель {{ .ReviewerModel }}.
Стадия: proposal.

Твоя задача: проверить артефакты OpenSpec change, указанные в TARGETS, и вернуть СТРОГО JSON по схеме SCHEMA. Не пиши вокруг JSON ничего. Не используй markdown-кодовые ограничители вокруг JSON.

CATEGORIES (закрытый enum, выбирай только из этого списка):
{{- range .Categories }}
- {{ . }}
{{- end }}

SEVERITY (закрытый enum):
- blocker — нельзя мёржить как есть
- major — нужно исправить до мёржа
- minor — стоит починить, но не блокирует
- nit — косметика

ПРАВИЛА fix_prompt: каждый prompt должен быть самодостаточным и исполняемым. Шаблон:

Контекст: [файл и строки].
Проблема: [одно предложение].
Задача: [одно предложение, что должно стать].
Ограничения: [правила из architecture.md / openspec / категория].
Acceptance: [как проверить, что починено].

SCHEMA:
{
  "summary": {
    "verdict": "ship-ready | needs-work | blocked",
    "walkthrough": "string (markdown)",
    "stats": {"findings": <int>, "by_severity": {"blocker": <int>, "major": <int>, "minor": <int>, "nit": <int>}}
  },
  "findings": [
    {
      "id": "F1",
      "category": "<one of CATEGORIES>",
      "severity": "blocker | major | minor | nit",
      "file": "string",
      "line_start": <int> | null,
      "line_end": <int> | null,
      "title": "string",
      "message": "string (markdown)",
      "fix_prompt": "string"
    }
  ]
}

TARGETS:
{{- range .Targets }}

=== {{ .Path }}{{ if .Truncated }} (TRUNCATED){{ end }} ===
{{ .Content }}
{{- end }}
```

- [ ] **Step 2: Write apply_review.tmpl**

Same structure but role text is "ревьюер реализации, проверяющий что код соответствует принятой спеке и не нарушает architecture.md", and uses Apply categories. Use Go template; same `.ProducerBy`/`.Categories`/`.Targets` fields.

- [ ] **Step 3: Write archive_review.tmpl**

Same structure for Archive.

- [ ] **Step 4: Write prompt.go**

```go
// internal/reviewrunner/prompt.go
package reviewrunner

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"orchv3/internal/agentmeta"
)

//go:embed prompts/*.tmpl
var defaultPromptFS embed.FS

type PromptInput struct {
	Stage          agentmeta.Stage
	ProducerBy     string
	ProducerModel  string
	ReviewerBy     string
	ReviewerModel  string
	Categories     []string
	Targets        []Target
}

func RenderPrompt(in PromptInput, overrideDir string) (string, error) {
	profile, err := ProfileFor(in.Stage)
	if err != nil {
		return "", err
	}

	tmplBytes, err := loadTemplate(profile.PromptName, overrideDir)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New(profile.PromptName).Parse(string(tmplBytes))
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", profile.PromptName, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, in); err != nil {
		return "", fmt.Errorf("execute template %s: %w", profile.PromptName, err)
	}
	return buf.String(), nil
}

func loadTemplate(name string, overrideDir string) ([]byte, error) {
	if overrideDir != "" {
		path := filepath.Join(overrideDir, name)
		if data, err := os.ReadFile(path); err == nil {
			return data, nil
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read override template %s: %w", path, err)
		}
	}
	return defaultPromptFS.ReadFile("prompts/" + name)
}
```

- [ ] **Step 5: Write prompt_test.go**

```go
package reviewrunner

import (
	"strings"
	"testing"

	"orchv3/internal/agentmeta"
)

func TestRenderPromptIncludesProducerReviewerStageAndTargets(t *testing.T) {
	got, err := RenderPrompt(PromptInput{
		Stage: agentmeta.StageProposal,
		ProducerBy: "codex", ProducerModel: "gpt-5-codex",
		ReviewerBy: "claude", ReviewerModel: "claude-sonnet-4-6",
		Categories: []string{"requirement_unclear", "scenario_missing"},
		Targets: []Target{{Path: "a.md", Content: "hello"}},
	}, "")
	if err != nil { t.Fatalf("RenderPrompt: %v", err) }
	for _, want := range []string{"codex", "claude", "gpt-5-codex", "requirement_unclear", "a.md", "hello"} {
		if !strings.Contains(got, want) { t.Fatalf("rendered prompt missing %q:\n%s", want, got) }
	}
}

func TestRenderPromptForUnknownStageFails(t *testing.T) {
	_, err := RenderPrompt(PromptInput{Stage: agentmeta.Stage("bogus")}, "")
	if err == nil { t.Fatal("expected error for unknown stage") }
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/reviewrunner/... -run Prompt -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/reviewrunner/prompts/ internal/reviewrunner/prompt.go internal/reviewrunner/prompt_test.go
git commit -m "Add review prompt templates and renderer"
```

---

## Task 11: PR commenter — interface, formatter, gh impl, idempotency

**Files:**
- Create: `internal/reviewrunner/prcommenter/commenter.go`
- Create: `internal/reviewrunner/prcommenter/format.go`
- Create: `internal/reviewrunner/prcommenter/format_test.go`
- Create: `internal/reviewrunner/prcommenter/gh.go`
- Create: `internal/reviewrunner/prcommenter/gh_test.go`

Этот task большой; разбит на чёткие шаги.

- [ ] **Step 1: Write commenter.go interface**

```go
// internal/reviewrunner/prcommenter/commenter.go
package prcommenter

import (
	"context"

	"orchv3/internal/agentmeta"
	"orchv3/internal/reviewrunner/reviewparse"
)

type PostReviewInput struct {
	RepoOwner    string
	RepoName     string
	PRNumber     int
	HeadSHA      string
	Stage        agentmeta.Stage
	ReviewerSlot string
	Review       reviewparse.Review
	WalkthroughExtras []string // tripwires (e.g., "producer trailer absent")
}

type PostReviewResult struct {
	Skipped bool // true when an existing review with the same marker was found
}

type PRCommenter interface {
	PostReview(ctx context.Context, in PostReviewInput) (PostReviewResult, error)
}

func MarkerFor(reviewerSlot string, stage agentmeta.Stage, headSHA string) string {
	return "<!-- drop-forge-review-marker:" + reviewerSlot + ":" + string(stage) + ":" + headSHA + " -->"
}
```

- [ ] **Step 2: Write format.go and tests**

```go
// internal/reviewrunner/prcommenter/format.go
package prcommenter

import (
	"fmt"
	"strings"

	"orchv3/internal/reviewrunner/reviewparse"
)

func FormatSummaryBody(in PostReviewInput) string {
	var b strings.Builder
	b.WriteString(MarkerFor(in.ReviewerSlot, in.Stage, in.HeadSHA))
	b.WriteString("\n\n## 🤖 Review by ")
	b.WriteString(in.ReviewerSlot)
	b.WriteString(" (stage: ")
	b.WriteString(string(in.Stage))
	b.WriteString(")\n\n")

	stats := in.Review.Summary.Stats
	fmt.Fprintf(&b, "**Verdict:** %s · **Findings:** %d (🛑 %d · ⚠️ %d · 💡 %d · 🪶 %d)\n\n",
		in.Review.Summary.Verdict,
		stats.Findings,
		stats.BySeverity["blocker"],
		stats.BySeverity["major"],
		stats.BySeverity["minor"],
		stats.BySeverity["nit"],
	)
	b.WriteString("### Walkthrough\n")
	b.WriteString(in.Review.Summary.Walkthrough)
	b.WriteString("\n\n")

	if len(in.Review.Findings) > 0 {
		b.WriteString("### Findings\n")
		for _, f := range in.Review.Findings {
			fmt.Fprintf(&b, "- %s **%s** [%s] %s — %s\n",
				severityIcon(f.Severity), f.ID, f.Category, formatLineRef(f), f.Title,
			)
		}
		b.WriteString("\n")
	}

	if len(in.WalkthroughExtras) > 0 {
		b.WriteString("### Tripwires\n")
		for _, t := range in.WalkthroughExtras {
			fmt.Fprintf(&b, "- %s\n", t)
		}
	}
	return b.String()
}

func FormatInlineBody(reviewerSlot string, f reviewparse.Finding) string {
	return fmt.Sprintf(
		"%s **[review by %s · severity: %s · category: %s]**\n\n%s\n\n<details>\n<summary>🤖 Prompt for AI Agent</summary>\n\n%s\n</details>\n",
		severityIcon(f.Severity), reviewerSlot, f.Severity, f.Category, f.Message, f.FixPrompt,
	)
}

func severityIcon(s reviewparse.Severity) string {
	switch s {
	case reviewparse.SeverityBlocker: return "🛑"
	case reviewparse.SeverityMajor:   return "⚠️"
	case reviewparse.SeverityMinor:   return "💡"
	case reviewparse.SeverityNit:     return "🪶"
	}
	return "•"
}

func formatLineRef(f reviewparse.Finding) string {
	if f.LineStart == nil {
		return f.File
	}
	if f.LineEnd != nil && *f.LineEnd != *f.LineStart {
		return fmt.Sprintf("%s:%d-%d", f.File, *f.LineStart, *f.LineEnd)
	}
	return fmt.Sprintf("%s:%d", f.File, *f.LineStart)
}
```

`format_test.go`:

```go
package prcommenter

import (
	"strings"
	"testing"

	"orchv3/internal/agentmeta"
	"orchv3/internal/reviewrunner/reviewparse"
)

func TestFormatSummaryBodyIncludesMarkerVerdictAndFindings(t *testing.T) {
	ls, le := 10, 14
	in := PostReviewInput{
		RepoOwner: "x", RepoName: "y", PRNumber: 42,
		HeadSHA: "abc123", Stage: agentmeta.StageProposal, ReviewerSlot: "codex",
		Review: reviewparse.Review{
			Summary: reviewparse.Summary{
				Verdict: reviewparse.VerdictNeedsWork,
				Walkthrough: "wt",
				Stats: reviewparse.Stats{Findings: 1, BySeverity: map[string]int{"blocker":0,"major":1,"minor":0,"nit":0}},
			},
			Findings: []reviewparse.Finding{{
				ID: "F1", Category: "scenario_missing", Severity: reviewparse.SeverityMajor,
				File: "openspec/x.md", LineStart: &ls, LineEnd: &le,
				Title: "Missing scenario", Message: "msg", FixPrompt: "fp",
			}},
		},
	}
	body := FormatSummaryBody(in)
	for _, want := range []string{
		"<!-- drop-forge-review-marker:codex:proposal:abc123 -->",
		"Review by codex",
		"needs-work", "F1", "scenario_missing", "openspec/x.md:10-14",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("summary missing %q:\n%s", want, body)
		}
	}
}

func TestFormatInlineBodyHasFixPromptDetailsBlock(t *testing.T) {
	ls := 5
	body := FormatInlineBody("codex", reviewparse.Finding{
		ID: "F1", Category: "bug", Severity: reviewparse.SeverityBlocker,
		File: "x.go", LineStart: &ls, Title: "t", Message: "m", FixPrompt: "do this",
	})
	if !strings.Contains(body, "🤖 Prompt for AI Agent") || !strings.Contains(body, "do this") {
		t.Fatalf("inline body missing details/fix:\n%s", body)
	}
	if !strings.Contains(body, "[review by codex · severity: blocker · category: bug]") {
		t.Fatalf("inline body missing review prefix:\n%s", body)
	}
}
```

- [ ] **Step 3: Write gh.go skeleton**

```go
// internal/reviewrunner/prcommenter/gh.go
package prcommenter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"orchv3/internal/commandrunner"
	"orchv3/internal/steplog"
)

type GHPostReviewCommenter struct {
	Command   commandrunner.Runner
	GHPath    string
	Service   string
	LogWriter func() (stdout, stderr interface{ Write([]byte) (int, error) })
}

type ghReview struct {
	Body string `json:"body"`
}

func (c GHPostReviewCommenter) PostReview(ctx context.Context, in PostReviewInput) (PostReviewResult, error) {
	exists, err := c.markerExists(ctx, in)
	if err != nil {
		return PostReviewResult{}, fmt.Errorf("check existing review marker: %w", err)
	}
	if exists {
		return PostReviewResult{Skipped: true}, nil
	}

	payload := buildPayload(in)
	body, err := json.Marshal(payload)
	if err != nil {
		return PostReviewResult{}, fmt.Errorf("encode review payload: %w", err)
	}

	endpoint := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", in.RepoOwner, in.RepoName, in.PRNumber)
	stdout := steplog.New(nil).LineWriter("github-review")
	defer stdout.Flush()

	if err := c.Command.Run(ctx, commandrunner.Command{
		Name:  c.GHPath,
		Args:  []string{"api", "-X", "POST", endpoint, "--input", "-"},
		Stdin: bytes.NewReader(body),
	}); err != nil {
		return PostReviewResult{}, fmt.Errorf("gh api POST review: %w", err)
	}
	return PostReviewResult{}, nil
}

func (c GHPostReviewCommenter) markerExists(ctx context.Context, in PostReviewInput) (bool, error) {
	endpoint := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", in.RepoOwner, in.RepoName, in.PRNumber)
	var out bytes.Buffer
	if err := c.Command.Run(ctx, commandrunner.Command{
		Name:   c.GHPath,
		Args:   []string{"api", endpoint, "--paginate"},
		Stdout: &out,
	}); err != nil {
		return false, fmt.Errorf("gh api GET reviews: %w", err)
	}
	var reviews []ghReview
	dec := json.NewDecoder(strings.NewReader(out.String()))
	for dec.More() {
		var batch []ghReview
		if err := dec.Decode(&batch); err != nil {
			return false, fmt.Errorf("decode reviews JSON: %w", err)
		}
		reviews = append(reviews, batch...)
	}
	marker := MarkerFor(in.ReviewerSlot, in.Stage, in.HeadSHA)
	for _, r := range reviews {
		if strings.Contains(r.Body, marker) { return true, nil }
	}
	return false, nil
}

func buildPayload(in PostReviewInput) map[string]interface{} {
	body := FormatSummaryBody(in)
	comments := []map[string]interface{}{}
	for _, f := range in.Review.Findings {
		if f.LineStart == nil { continue }
		c := map[string]interface{}{
			"path": f.File,
			"line": *f.LineStart,
			"side": "RIGHT",
			"body": FormatInlineBody(in.ReviewerSlot, f),
		}
		if f.LineEnd != nil && *f.LineEnd != *f.LineStart {
			c["start_line"] = *f.LineStart
			c["line"] = *f.LineEnd
			c["start_side"] = "RIGHT"
		}
		comments = append(comments, c)
	}
	return map[string]interface{}{
		"commit_id": in.HeadSHA,
		"event":     "COMMENT",
		"body":      body,
		"comments":  comments,
	}
}
```

- [ ] **Step 4: Write gh_test.go**

```go
package prcommenter

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"orchv3/internal/agentmeta"
	"orchv3/internal/commandrunner"
	"orchv3/internal/reviewrunner/reviewparse"
)

type fakeRunner struct {
	commands []commandrunner.Command
	getOut   string
	postErr  error
}

func (f *fakeRunner) Run(_ context.Context, c commandrunner.Command) error {
	f.commands = append(f.commands, c)
	if isGet(c) {
		if c.Stdout != nil { _, _ = io.WriteString(c.Stdout.(io.Writer), f.getOut) }
		return nil
	}
	return f.postErr
}

func isGet(c commandrunner.Command) bool {
	for _, a := range c.Args { if a == "POST" { return false } }
	return true
}

func TestPostReviewSkipsWhenMarkerAlreadyPresent(t *testing.T) {
	marker := MarkerFor("codex", agentmeta.StageProposal, "deadbeef")
	resp, _ := json.Marshal([]ghReview{{Body: marker + "\n\nold body"}})
	fake := &fakeRunner{getOut: string(resp)}
	c := GHPostReviewCommenter{Command: fake, GHPath: "gh"}

	res, err := c.PostReview(context.Background(), PostReviewInput{
		RepoOwner: "o", RepoName: "r", PRNumber: 1,
		HeadSHA: "deadbeef", Stage: agentmeta.StageProposal, ReviewerSlot: "codex",
		Review: reviewparse.Review{Summary: reviewparse.Summary{Verdict: reviewparse.VerdictShipReady}},
	})
	if err != nil { t.Fatalf("PostReview: %v", err) }
	if !res.Skipped { t.Fatalf("expected Skipped=true, got %+v", res) }
	if len(fake.commands) != 1 { t.Fatalf("commands=%d, want 1 (only GET)", len(fake.commands)) }
}

func TestPostReviewSendsAtomicPOSTWithSummaryAndInlineComments(t *testing.T) {
	fake := &fakeRunner{getOut: "[]"}
	ls := 14
	in := PostReviewInput{
		RepoOwner: "o", RepoName: "r", PRNumber: 7,
		HeadSHA: "feedface", Stage: agentmeta.StageApply, ReviewerSlot: "codex",
		Review: reviewparse.Review{
			Summary: reviewparse.Summary{Verdict: reviewparse.VerdictNeedsWork, Stats: reviewparse.Stats{BySeverity: map[string]int{"major":1}}},
			Findings: []reviewparse.Finding{{
				ID: "F1", Category: "bug", Severity: reviewparse.SeverityMajor,
				File: "internal/x.go", LineStart: &ls, Title: "t", Message: "m", FixPrompt: "fp",
			}},
		},
	}
	c := GHPostReviewCommenter{Command: fake, GHPath: "gh"}
	if _, err := c.PostReview(context.Background(), in); err != nil { t.Fatalf("PostReview: %v", err) }
	if len(fake.commands) != 2 { t.Fatalf("commands=%d, want 2 (GET+POST)", len(fake.commands)) }
	post := fake.commands[1]
	if post.Stdin == nil { t.Fatal("POST missing stdin payload") }
	body, _ := io.ReadAll(post.Stdin)
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil { t.Fatalf("decode payload: %v", err) }
	if payload["event"] != "COMMENT" { t.Fatalf("event=%v", payload["event"]) }
	if payload["commit_id"] != "feedface" { t.Fatalf("commit_id=%v", payload["commit_id"]) }
	if !strings.Contains(payload["body"].(string), "drop-forge-review-marker:codex:apply:feedface") {
		t.Fatalf("body missing marker: %v", payload["body"])
	}
	comments, ok := payload["comments"].([]interface{})
	if !ok || len(comments) != 1 { t.Fatalf("comments=%v", payload["comments"]) }
}
```

- [ ] **Step 5: Run all prcommenter tests**

Run: `go test ./internal/reviewrunner/prcommenter/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/reviewrunner/prcommenter/
git commit -m "Add PR review commenter with idempotent gh api wiring"
```

---

## Task 12: Reviewer slot selection

**Files:**
- Create: `internal/reviewrunner/reviewer.go`
- Create: `internal/reviewrunner/reviewer_test.go`

Цель: чистая функция, которая по producer trailer'у и `ReviewRunnerConfig` выдаёт reviewer slot.

- [ ] **Step 1: Write failing tests**

```go
// internal/reviewrunner/reviewer_test.go
package reviewrunner

import (
	"errors"
	"testing"

	"orchv3/internal/agentmeta"
	"orchv3/internal/config"
)

func TestSelectReviewerProducerIsPrimaryReturnsSecondary(t *testing.T) {
	cfg := config.ReviewRunnerConfig{PrimarySlot: "codex", SecondarySlot: "claude", PrimaryModel: "g", SecondaryModel: "c", PrimaryExecutorPath: "/p", SecondaryExecutorPath: "/c"}
	got, err := SelectReviewer(cfg, agentmeta.Producer{By: "codex"})
	if err != nil { t.Fatalf("SelectReviewer: %v", err) }
	if got.Slot != "claude" || got.Model != "c" || got.ExecutorPath != "/c" {
		t.Fatalf("got %+v", got)
	}
}

func TestSelectReviewerProducerIsSecondaryReturnsPrimary(t *testing.T) {
	cfg := config.ReviewRunnerConfig{PrimarySlot: "codex", SecondarySlot: "claude", PrimaryModel: "g", SecondaryModel: "c", PrimaryExecutorPath: "/p", SecondaryExecutorPath: "/c"}
	got, _ := SelectReviewer(cfg, agentmeta.Producer{By: "claude"})
	if got.Slot != "codex" {
		t.Fatalf("got slot=%s, want codex", got.Slot)
	}
}

func TestSelectReviewerWithoutProducerFallsBackToSecondary(t *testing.T) {
	cfg := config.ReviewRunnerConfig{PrimarySlot: "codex", SecondarySlot: "claude", PrimaryModel: "g", SecondaryModel: "c", PrimaryExecutorPath: "/p", SecondaryExecutorPath: "/c"}
	got, err := SelectReviewer(cfg, agentmeta.Producer{}) // empty trailer
	if err != nil { t.Fatalf("err = %v", err) }
	if got.Slot != "claude" {
		t.Fatalf("got slot=%s, want secondary fallback", got.Slot)
	}
	if !got.ProducerUnknown { t.Fatal("expected ProducerUnknown=true") }
}

func TestSelectReviewerUnknownProducerSlotIsConfigError(t *testing.T) {
	cfg := config.ReviewRunnerConfig{PrimarySlot: "codex", SecondarySlot: "claude", PrimaryModel: "g", SecondaryModel: "c", PrimaryExecutorPath: "/p", SecondaryExecutorPath: "/c"}
	_, err := SelectReviewer(cfg, agentmeta.Producer{By: "bardo"})
	if !errors.Is(err, ErrUnknownProducerSlot) {
		t.Fatalf("err = %v, want ErrUnknownProducerSlot", err)
	}
}
```

- [ ] **Step 2: Implement reviewer.go**

```go
// internal/reviewrunner/reviewer.go
package reviewrunner

import (
	"errors"
	"fmt"

	"orchv3/internal/agentmeta"
	"orchv3/internal/config"
)

type Reviewer struct {
	Slot            string
	Model           string
	ExecutorPath    string
	ProducerUnknown bool
}

var ErrUnknownProducerSlot = errors.New("producer slot not in REVIEW_ROLE_PRIMARY or REVIEW_ROLE_SECONDARY")

func SelectReviewer(cfg config.ReviewRunnerConfig, producer agentmeta.Producer) (Reviewer, error) {
	if producer.By == "" {
		return Reviewer{
			Slot: cfg.SecondarySlot, Model: cfg.SecondaryModel,
			ExecutorPath: cfg.SecondaryExecutorPath, ProducerUnknown: true,
		}, nil
	}
	switch producer.By {
	case cfg.PrimarySlot:
		return Reviewer{Slot: cfg.SecondarySlot, Model: cfg.SecondaryModel, ExecutorPath: cfg.SecondaryExecutorPath}, nil
	case cfg.SecondarySlot:
		return Reviewer{Slot: cfg.PrimarySlot, Model: cfg.PrimaryModel, ExecutorPath: cfg.PrimaryExecutorPath}, nil
	}
	return Reviewer{}, fmt.Errorf("%w: producer=%q primary=%q secondary=%q",
		ErrUnknownProducerSlot, producer.By, cfg.PrimarySlot, cfg.SecondarySlot)
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/reviewrunner/... -run SelectReviewer -v`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/reviewrunner/reviewer.go internal/reviewrunner/reviewer_test.go
git commit -m "Add reviewer slot selection by producer trailer"
```

---

## Task 13: ReviewRunner — agent executor interface and Codex impl

**Files:**
- Create: `internal/reviewrunner/agent_executor.go`
- Create: `internal/reviewrunner/codex_executor.go`

Цель: тот же паттерн, что в `applyrunner` — `AgentExecutor` interface, `CodexCLIExecutor` реализация, чтобы `ReviewRunner` мог инжектить fake в тестах.

- [ ] **Step 1: Write agent_executor.go**

```go
// internal/reviewrunner/agent_executor.go
package reviewrunner

import (
	"context"
	"io"
)

type AgentExecutionInput struct {
	Prompt   string
	CloneDir string
	TempDir  string
	Stdout   io.Writer
	Stderr   io.Writer
}

type AgentExecutionResult struct {
	FinalMessage string
}

type AgentExecutor interface {
	Run(ctx context.Context, input AgentExecutionInput) (AgentExecutionResult, error)
}
```

- [ ] **Step 2: Write codex_executor.go**

Read `internal/proposalrunner/agent_executor.go` and `codex_executor.go` first to mirror invocation. Then:

```go
// internal/reviewrunner/codex_executor.go
package reviewrunner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"orchv3/internal/commandrunner"
	"orchv3/internal/steplog"
)

type CodexCLIExecutor struct {
	Command      commandrunner.Runner
	CodexPath    string
	Model        string
	Service      string
}

func (e CodexCLIExecutor) Run(ctx context.Context, input AgentExecutionInput) (AgentExecutionResult, error) {
	logger := steplog.NewWithService(input.Stdout, e.Service)
	finalPath := filepath.Join(input.TempDir, "review-final.txt")

	args := []string{
		"exec", "--json",
		"--sandbox", "danger-full-access",
		"--output-last-message", finalPath,
		"--cd", input.CloneDir,
		"--model", e.Model,
		"-",
	}
	logger.Infof("codex", "%s %s", e.CodexPath, strings.Join(args, " "))

	if err := e.Command.Run(ctx, commandrunner.Command{
		Name:   e.CodexPath,
		Args:   args,
		Stdin:  bytes.NewReader([]byte(input.Prompt)),
		Stdout: input.Stdout,
		Stderr: input.Stderr,
	}); err != nil {
		return AgentExecutionResult{}, fmt.Errorf("codex exec: %w", err)
	}
	final, err := os.ReadFile(finalPath)
	if err != nil {
		return AgentExecutionResult{}, fmt.Errorf("read final review message: %w", err)
	}
	return AgentExecutionResult{FinalMessage: strings.TrimSpace(string(final))}, nil
}
```

- [ ] **Step 3: Build to verify compiles**

Run: `go build ./internal/reviewrunner/...`
Expected: success (no test code yet for executor — it's exercised through ReviewRunner tests in Task 14).

- [ ] **Step 4: Commit**

```bash
git add internal/reviewrunner/agent_executor.go internal/reviewrunner/codex_executor.go
git commit -m "Add review agent executor interface and codex implementation"
```

---

## Task 14: ReviewRunner.Run() — orchestration

**Files:**
- Create: `internal/reviewrunner/runner.go`
- Create: `internal/reviewrunner/runner_test.go`

Самая большая задача. Орekestrates: clone → resolve change path → read trailer → select reviewer → collect targets → render prompt → exec → parse (with one repair retry) → publish → return.

- [ ] **Step 1: Write runner.go skeleton**

```go
// internal/reviewrunner/runner.go
package reviewrunner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"orchv3/internal/agentmeta"
	"orchv3/internal/commandrunner"
	"orchv3/internal/config"
	"orchv3/internal/reviewrunner/prcommenter"
	"orchv3/internal/reviewrunner/reviewparse"
	"orchv3/internal/steplog"
)

const defaultTempPattern = "orchv3-review-*"

type Runner struct {
	Config       config.ReviewRunnerConfig
	ProposalCfg  config.ProposalRunnerConfig
	Command      commandrunner.Runner
	Executors    map[string]AgentExecutor // keyed by reviewer slot name
	Commenter    prcommenter.PRCommenter
	Service      string
	Stdout       io.Writer
	Stderr       io.Writer
	MkdirTemp    func(dir, pattern string) (string, error)
	RemoveAll    func(path string) error
}

type ReviewInput struct {
	Stage      agentmeta.Stage
	Identifier string
	Title      string
	BranchName string
	PRNumber   int
	RepoOwner  string
	RepoName   string
	PRURL      string
}

func (in ReviewInput) validate() error {
	if in.Stage == "" {
		return errors.New("review input stage must not be empty")
	}
	if strings.TrimSpace(in.BranchName) == "" {
		return errors.New("review input branch name must not be empty")
	}
	if in.PRNumber == 0 {
		return errors.New("review input PR number must not be zero")
	}
	return nil
}

type Result struct {
	Skipped bool
}

func (r *Runner) Run(ctx context.Context, in ReviewInput) (Result, error) {
	if err := in.validate(); err != nil {
		return Result{}, err
	}
	stdout := writerOrDiscard(r.Stdout)
	stderr := writerOrDiscard(r.Stderr)
	logger := steplog.NewWithService(stdout, r.Service)
	logger.Infof("review", "start stage=%s pr=%d branch=%s", in.Stage, in.PRNumber, in.BranchName)

	tempDir, err := r.mkdirTemp("", defaultTempPattern)
	if err != nil { return Result{}, fmt.Errorf("create temp dir: %w", err) }
	defer func() {
		if r.ProposalCfg.CleanupTemp { _ = r.removeAll(tempDir) }
	}()
	cloneDir := filepath.Join(tempDir, "repo")

	if err := r.gitClone(ctx, in.BranchName, cloneDir, stdout, stderr); err != nil {
		return Result{}, err
	}

	headSHA, message, err := r.readHead(ctx, cloneDir, stdout, stderr)
	if err != nil { return Result{}, err }

	producer, trailerErr := agentmeta.ParseTrailer(message)
	if trailerErr != nil && !errors.Is(trailerErr, agentmeta.ErrTrailerNotFound) {
		return Result{}, fmt.Errorf("parse producer trailer: %w", trailerErr)
	}
	reviewer, err := SelectReviewer(r.Config, producer)
	if err != nil { return Result{}, fmt.Errorf("select reviewer: %w", err) }

	exec, ok := r.Executors[reviewer.Slot]
	if !ok {
		return Result{}, fmt.Errorf("no agent executor registered for slot %q", reviewer.Slot)
	}

	changePath, _ := r.detectChangePath(ctx, cloneDir, in.Stage, stdout, stderr)
	diff, _ := r.gitDiff(ctx, cloneDir, stdout, stderr)

	targets, err := CollectTargets(TargetInput{
		Stage: in.Stage, CloneDir: cloneDir, MaxBytes: r.Config.MaxContextBytes,
		ChangePath: changePath, Diff: diff,
	})
	if err != nil { return Result{}, fmt.Errorf("collect targets: %w", err) }

	profile := MustProfile(in.Stage)
	categories := categorySliceFromMap(profile.Categories)

	prompt, err := RenderPrompt(PromptInput{
		Stage: in.Stage,
		ProducerBy: producer.By, ProducerModel: producer.Model,
		ReviewerBy: reviewer.Slot, ReviewerModel: reviewer.Model,
		Categories: categories,
		Targets: targets,
	}, r.Config.PromptDir)
	if err != nil { return Result{}, fmt.Errorf("render prompt: %w", err) }

	review, err := r.executeWithRepair(ctx, exec, prompt, in.Stage, cloneDir, tempDir, stdout, stderr)
	if err != nil { return Result{}, err }

	extras := []string{}
	if reviewer.ProducerUnknown {
		extras = append(extras, "Producer trailer absent — reviewer chosen by REVIEW_ROLE_SECONDARY.")
	}
	for _, target := range targets {
		if target.Truncated { extras = append(extras, fmt.Sprintf("Target truncated: %s", target.Path)) }
	}

	res, err := r.Commenter.PostReview(ctx, prcommenter.PostReviewInput{
		RepoOwner: in.RepoOwner, RepoName: in.RepoName, PRNumber: in.PRNumber,
		HeadSHA: headSHA, Stage: in.Stage, ReviewerSlot: reviewer.Slot,
		Review: review, WalkthroughExtras: extras,
	})
	if err != nil { return Result{}, fmt.Errorf("post review: %w", err) }
	if res.Skipped {
		logger.Infof("review", "skipped_idempotent stage=%s sha=%s", in.Stage, headSHA)
	} else {
		logger.Infof("review", "publish ok stage=%s findings=%d", in.Stage, len(review.Findings))
	}
	return Result{Skipped: res.Skipped}, nil
}

func (r *Runner) executeWithRepair(ctx context.Context, exec AgentExecutor, prompt string, stage agentmeta.Stage, cloneDir, tempDir string, stdout, stderr io.Writer) (reviewparse.Review, error) {
	result, err := exec.Run(ctx, AgentExecutionInput{Prompt: prompt, CloneDir: cloneDir, TempDir: tempDir, Stdout: stdout, Stderr: stderr})
	if err != nil { return reviewparse.Review{}, fmt.Errorf("agent review: %w", err) }
	review, parseErr := reviewparse.Parse([]byte(result.FinalMessage), stage)
	if parseErr == nil { return review, nil }
	if r.Config.ParseRepairRetries < 1 {
		return reviewparse.Review{}, fmt.Errorf("parse review JSON: %w", parseErr)
	}

	repairPrompt := fmt.Sprintf(
		"Твой предыдущий ответ невалиден. Ошибка: %v. Верни строго JSON по схеме без лишнего текста.\n\n--- предыдущий ответ ---\n%s",
		parseErr, result.FinalMessage,
	)
	result2, err := exec.Run(ctx, AgentExecutionInput{Prompt: repairPrompt, CloneDir: cloneDir, TempDir: tempDir, Stdout: stdout, Stderr: stderr})
	if err != nil { return reviewparse.Review{}, fmt.Errorf("agent review repair: %w", err) }
	review, parseErr = reviewparse.Parse([]byte(result2.FinalMessage), stage)
	if parseErr != nil { return reviewparse.Review{}, fmt.Errorf("parse review JSON after repair: %w", parseErr) }
	return review, nil
}

func (r *Runner) gitClone(ctx context.Context, branch, cloneDir string, stdout, stderr io.Writer) error {
	parent := filepath.Dir(cloneDir)
	if err := os.MkdirAll(parent, 0o755); err != nil { return fmt.Errorf("mkdir parent: %w", err) }
	return r.Command.Run(ctx, commandrunner.Command{
		Name: r.ProposalCfg.GitPath,
		Args: []string{"clone", "--branch", branch, r.ProposalCfg.RepositoryURL, cloneDir},
		Dir:  parent, Stdout: stdout, Stderr: stderr,
	})
}

func (r *Runner) readHead(ctx context.Context, cloneDir string, stdout, stderr io.Writer) (string, string, error) {
	var shaOut bytes.Buffer
	if err := r.Command.Run(ctx, commandrunner.Command{
		Name: r.ProposalCfg.GitPath, Args: []string{"rev-parse", "HEAD"}, Dir: cloneDir, Stdout: &shaOut, Stderr: stderr,
	}); err != nil { return "", "", fmt.Errorf("git rev-parse: %w", err) }

	var msgOut bytes.Buffer
	if err := r.Command.Run(ctx, commandrunner.Command{
		Name: r.ProposalCfg.GitPath, Args: []string{"log", "-1", "--format=%B"}, Dir: cloneDir, Stdout: &msgOut, Stderr: stderr,
	}); err != nil { return "", "", fmt.Errorf("git log: %w", err) }
	return strings.TrimSpace(shaOut.String()), msgOut.String(), nil
}

func (r *Runner) gitDiff(ctx context.Context, cloneDir string, stdout, stderr io.Writer) (string, error) {
	var out bytes.Buffer
	err := r.Command.Run(ctx, commandrunner.Command{
		Name: r.ProposalCfg.GitPath, Args: []string{"diff", r.ProposalCfg.BaseBranch + "...HEAD"}, Dir: cloneDir, Stdout: &out, Stderr: stderr,
	})
	return out.String(), err
}

func (r *Runner) detectChangePath(ctx context.Context, cloneDir string, stage agentmeta.Stage, stdout, stderr io.Writer) (string, error) {
	var out bytes.Buffer
	err := r.Command.Run(ctx, commandrunner.Command{
		Name: r.ProposalCfg.GitPath,
		Args: []string{"diff", "--name-only", r.ProposalCfg.BaseBranch + "...HEAD"},
		Dir:  cloneDir, Stdout: &out, Stderr: stderr,
	})
	if err != nil { return "", err }
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if strings.HasPrefix(line, "openspec/changes/") {
			parts := strings.SplitN(line, "/", 4)
			if len(parts) >= 3 {
				return strings.Join(parts[:3], "/"), nil
			}
		}
	}
	return "", nil
}

func (r *Runner) mkdirTemp(dir, pattern string) (string, error) {
	if r.MkdirTemp != nil { return r.MkdirTemp(dir, pattern) }
	return os.MkdirTemp(dir, pattern)
}
func (r *Runner) removeAll(path string) error {
	if r.RemoveAll != nil { return r.RemoveAll(path) }
	return os.RemoveAll(path)
}

func categorySliceFromMap(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m { out = append(out, k) }
	return out
}

func writerOrDiscard(w io.Writer) io.Writer { if w == nil { return io.Discard }; return w }
```

- [ ] **Step 2: Write runner_test.go covering happy path, repair, parse fail twice, missing trailer fallback, idempotent skip**

```go
// internal/reviewrunner/runner_test.go
package reviewrunner

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"orchv3/internal/agentmeta"
	"orchv3/internal/commandrunner"
	"orchv3/internal/config"
	"orchv3/internal/reviewrunner/prcommenter"
	"orchv3/internal/reviewrunner/reviewparse"
)

type fakeRun struct {
	stdout, stderr string
	err            error
}

type fakeCmd struct {
	commands []commandrunner.Command
	plans    []fakeRun
	idx      int
}

func (f *fakeCmd) Run(_ context.Context, c commandrunner.Command) error {
	f.commands = append(f.commands, c)
	if f.idx >= len(f.plans) {
		return nil
	}
	plan := f.plans[f.idx]
	f.idx++
	if c.Stdout != nil && plan.stdout != "" { _, _ = io.WriteString(c.Stdout.(io.Writer), plan.stdout) }
	if c.Stderr != nil && plan.stderr != "" { _, _ = io.WriteString(c.Stderr.(io.Writer), plan.stderr) }
	return plan.err
}

type fakeExec struct {
	responses []string
	calls     int
	err       error
}

func (f *fakeExec) Run(_ context.Context, _ AgentExecutionInput) (AgentExecutionResult, error) {
	if f.err != nil { return AgentExecutionResult{}, f.err }
	if f.calls >= len(f.responses) { return AgentExecutionResult{}, errors.New("no more fake responses") }
	r := f.responses[f.calls]
	f.calls++
	return AgentExecutionResult{FinalMessage: r}, nil
}

type fakeCommenter struct {
	called bool
	in     prcommenter.PostReviewInput
	res    prcommenter.PostReviewResult
	err    error
}

func (f *fakeCommenter) PostReview(_ context.Context, in prcommenter.PostReviewInput) (prcommenter.PostReviewResult, error) {
	f.called = true; f.in = in
	return f.res, f.err
}

const validReviewJSON = `{
  "summary": {"verdict":"ship-ready","walkthrough":"ok","stats":{"findings":0,"by_severity":{"blocker":0,"major":0,"minor":0,"nit":0}}},
  "findings": []
}`

func newRunnerForTest(t *testing.T, cmd commandrunner.Runner, exec AgentExecutor, comm prcommenter.PRCommenter) *Runner {
	tempBase := t.TempDir()
	return &Runner{
		Config: config.ReviewRunnerConfig{
			PrimarySlot: "codex", SecondarySlot: "claude",
			PrimaryModel: "g", SecondaryModel: "c",
			PrimaryExecutorPath: "/p", SecondaryExecutorPath: "/c",
			MaxContextBytes: 1 << 16, ParseRepairRetries: 1,
		},
		ProposalCfg: config.ProposalRunnerConfig{
			RepositoryURL: "https://example/repo", BaseBranch: "main", RemoteName: "origin",
			BranchPrefix: "p", PRTitlePrefix: "P:", GitPath: "git", CodexPath: "codex", GHPath: "gh",
		},
		Command: cmd,
		Executors: map[string]AgentExecutor{"claude": exec},
		Commenter: comm,
		Service: "orchv3",
		Stdout: io.Discard, Stderr: io.Discard,
		MkdirTemp: func(dir, pattern string) (string, error) { return filepath.Join(tempBase, "rev"), nil },
		RemoveAll: func(path string) error { return os.RemoveAll(path) },
	}
}

func TestRunHappyPathPublishesReviewAndReturnsResult(t *testing.T) {
	cmd := &fakeCmd{plans: []fakeRun{
		{}, // git clone
		{stdout: "deadbeef\n"},                                      // git rev-parse HEAD
		{stdout: "subject\n\nProduced-By: codex\nProduced-Model: gpt-5\nProduced-Stage: proposal\n"}, // git log -1 --format=%B
		{stdout: ""},                                                  // git diff --name-only
		{stdout: ""},                                                  // git diff
	}}
	exec := &fakeExec{responses: []string{validReviewJSON}}
	comm := &fakeCommenter{}
	r := newRunnerForTest(t, cmd, exec, comm)

	res, err := r.Run(context.Background(), ReviewInput{
		Stage: agentmeta.StageProposal, BranchName: "feature/x", PRNumber: 1,
		RepoOwner: "o", RepoName: "p",
	})
	if err != nil { t.Fatalf("Run: %v", err) }
	if res.Skipped { t.Fatal("expected not skipped") }
	if !comm.called { t.Fatal("commenter not called") }
	if comm.in.HeadSHA != "deadbeef" { t.Fatalf("HeadSHA=%q", comm.in.HeadSHA) }
	if comm.in.ReviewerSlot != "claude" { t.Fatalf("ReviewerSlot=%q", comm.in.ReviewerSlot) }
}

func TestRunMissingTrailerFallsBackAndAddsTripwire(t *testing.T) {
	cmd := &fakeCmd{plans: []fakeRun{
		{}, {stdout: "abc\n"}, {stdout: "subject only\n"}, {}, {},
	}}
	exec := &fakeExec{responses: []string{validReviewJSON}}
	comm := &fakeCommenter{}
	r := newRunnerForTest(t, cmd, exec, comm)

	if _, err := r.Run(context.Background(), ReviewInput{
		Stage: agentmeta.StageProposal, BranchName: "x", PRNumber: 1, RepoOwner: "o", RepoName: "p",
	}); err != nil { t.Fatalf("Run: %v", err) }
	found := false
	for _, e := range comm.in.WalkthroughExtras {
		if contains(e, "Producer trailer absent") { found = true }
	}
	if !found { t.Fatalf("expected tripwire, got %v", comm.in.WalkthroughExtras) }
}

func TestRunRepairsInvalidJSONOnce(t *testing.T) {
	cmd := &fakeCmd{plans: []fakeRun{
		{}, {stdout: "abc\n"}, {stdout: "subject\n\nProduced-By: codex\nProduced-Model: x\nProduced-Stage: proposal\n"}, {}, {},
	}}
	exec := &fakeExec{responses: []string{"not-json", validReviewJSON}}
	comm := &fakeCommenter{}
	r := newRunnerForTest(t, cmd, exec, comm)

	if _, err := r.Run(context.Background(), ReviewInput{
		Stage: agentmeta.StageProposal, BranchName: "x", PRNumber: 1, RepoOwner: "o", RepoName: "p",
	}); err != nil { t.Fatalf("Run: %v", err) }
	if exec.calls != 2 { t.Fatalf("exec.calls=%d, want 2 (initial + repair)", exec.calls) }
	if !comm.called { t.Fatal("commenter not called after repair") }
}

func TestRunReturnsErrorWhenRepairAlsoInvalid(t *testing.T) {
	cmd := &fakeCmd{plans: []fakeRun{{}, {stdout: "abc\n"}, {stdout: "subject\n\nProduced-By: codex\nProduced-Model: x\nProduced-Stage: proposal\n"}, {}, {}}}
	exec := &fakeExec{responses: []string{"not-json", "still-not-json"}}
	comm := &fakeCommenter{}
	r := newRunnerForTest(t, cmd, exec, comm)

	_, err := r.Run(context.Background(), ReviewInput{
		Stage: agentmeta.StageProposal, BranchName: "x", PRNumber: 1, RepoOwner: "o", RepoName: "p",
	})
	if err == nil { t.Fatal("expected error after repair failure") }
	if comm.called { t.Fatal("commenter must not be called when parse fails") }
}

func TestRunReportsSkippedWhenCommenterReportsSkip(t *testing.T) {
	cmd := &fakeCmd{plans: []fakeRun{{}, {stdout: "abc\n"}, {stdout: "subject\n\nProduced-By: codex\nProduced-Model: x\nProduced-Stage: proposal\n"}, {}, {}}}
	exec := &fakeExec{responses: []string{validReviewJSON}}
	comm := &fakeCommenter{res: prcommenter.PostReviewResult{Skipped: true}}
	r := newRunnerForTest(t, cmd, exec, comm)

	res, err := r.Run(context.Background(), ReviewInput{
		Stage: agentmeta.StageProposal, BranchName: "x", PRNumber: 1, RepoOwner: "o", RepoName: "p",
	})
	if err != nil { t.Fatalf("Run: %v", err) }
	if !res.Skipped { t.Fatalf("expected Skipped=true") }
}

func contains(s, sub string) bool {
	if len(sub) > len(s) { return false }
	for i := 0; i+len(sub) <= len(s); i++ { if s[i:i+len(sub)] == sub { return true } }
	return false
}

var _ = reviewparse.VerdictShipReady // avoid unused import warning if tests are pruned
```

- [ ] **Step 3: Run all reviewrunner tests**

Run: `go test ./internal/reviewrunner/...`
Expected: PASS for runner_test.go and all earlier tests.

- [ ] **Step 4: Commit**

```bash
git add internal/reviewrunner/runner.go internal/reviewrunner/runner_test.go
git commit -m "Implement review runner orchestration with repair retry"
```

---

## Task 15: CoreOrch — extend Config and BuildReviewInput

**Files:**
- Modify: `internal/coreorch/orchestrator.go:37-47` (extend Config)
- Modify: `internal/coreorch/orchestrator.go:362-404` (extend validate)
- Modify: `internal/coreorch/orchestrator.go` (add BuildReviewInput, ReviewRunner interface, processReviewTask)
- Modify: `internal/coreorch/orchestrator_test.go`

- [ ] **Step 1: Add ReviewRunner interface and Config fields**

```go
// internal/coreorch/orchestrator.go
import (
	// ... existing imports
	"orchv3/internal/reviewrunner"
)

type ReviewRunner interface {
	Run(ctx context.Context, input reviewrunner.ReviewInput) (reviewrunner.Result, error)
}

type Config struct {
	ReadyToProposeStateID         string
	ProposingInProgressStateID    string
	NeedProposalReviewStateID     string
	NeedProposalAIReviewStateID   string  // NEW
	ReadyToCodeStateID            string
	CodeInProgressStateID         string
	NeedCodeReviewStateID         string
	NeedCodeAIReviewStateID       string  // NEW
	ReadyToArchiveStateID         string
	ArchivingInProgressStateID    string
	NeedArchiveReviewStateID      string
	NeedArchiveAIReviewStateID    string  // NEW
	AIReviewEnabled               bool    // computed in cmd/orchv3 wiring
}
```

Add `ReviewRunner ReviewRunner` field on `Orchestrator` struct.

- [ ] **Step 2: Add BuildReviewInput**

Place near other builders:

```go
// internal/coreorch/orchestrator.go
import (
	"net/url"
	"strconv"
)

func BuildReviewInput(task taskmanager.Task, stage agentmeta.Stage) (reviewrunner.ReviewInput, error) {
	prURL, branchName := branchSource(task.PullRequests)
	if branchName == "" && prURL == "" {
		return reviewrunner.ReviewInput{}, fmt.Errorf("pull request branch source is missing")
	}
	owner, repo, number, err := parseGitHubPR(prURL)
	if err != nil { return reviewrunner.ReviewInput{}, fmt.Errorf("parse PR URL: %w", err) }
	title := strings.TrimSpace(task.Title)
	if title == "" { title = "Untitled task" }
	return reviewrunner.ReviewInput{
		Stage: stage, Identifier: strings.TrimSpace(task.Identifier),
		Title: title, BranchName: branchName, PRNumber: number,
		RepoOwner: owner, RepoName: repo, PRURL: prURL,
	}, nil
}

func parseGitHubPR(prURL string) (string, string, int, error) {
	u, err := url.Parse(prURL)
	if err != nil { return "", "", 0, err }
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 || parts[2] != "pull" {
		return "", "", 0, fmt.Errorf("unexpected PR URL path: %s", u.Path)
	}
	num, err := strconv.Atoi(parts[3])
	if err != nil { return "", "", 0, fmt.Errorf("parse PR number: %w", err) }
	return parts[0], parts[1], num, nil
}
```

Add `"orchv3/internal/agentmeta"` and `"orchv3/internal/reviewrunner"` imports if not yet present.

- [ ] **Step 3: Add processReviewTask method**

```go
// internal/coreorch/orchestrator.go

func (orch *Orchestrator) processReviewTask(ctx context.Context, logger steplog.Logger, task taskmanager.Task, stage agentmeta.Stage, targetState string) error {
	taskRef := taskReference(task)
	logger.Infof(module, "process %s ai-review task=%s identifier=%s", stage, task.ID, task.Identifier)
	in, err := BuildReviewInput(task, stage)
	if err != nil {
		logger.Errorf(module, "build review input %s: %v", taskRef, err)
		return fmt.Errorf("process %s ai-review %s: build input: %w", stage, taskRef, err)
	}
	if _, err := orch.ReviewRunner.Run(ctx, in); err != nil {
		logger.Errorf(module, "run %s ai-review %s: %v", stage, taskRef, err)
		return fmt.Errorf("process %s ai-review %s: run review: %w", stage, taskRef, err)
	}
	if err := orch.TaskManager.MoveTask(ctx, task.ID, targetState); err != nil {
		logger.Errorf(module, "move %s review task %s state=%s: %v", stage, taskRef, targetState, err)
		return fmt.Errorf("process %s ai-review %s: move to human review state %s: %w", stage, taskRef, targetState, err)
	}
	logger.Infof(module, "processed %s ai-review task=%s identifier=%s", stage, task.ID, task.Identifier)
	return nil
}
```

- [ ] **Step 4: Add three AI-review cases to RunProposalsOnce switch**

```go
// internal/coreorch/orchestrator.go inside RunProposalsOnce switch block:
case orch.Config.NeedProposalAIReviewStateID:
	if !orch.Config.AIReviewEnabled { goto skipDefault }
	if err := orch.processReviewTask(ctx, logger, task, agentmeta.StageProposal, orch.Config.NeedProposalReviewStateID); err != nil { return err }
case orch.Config.NeedCodeAIReviewStateID:
	if !orch.Config.AIReviewEnabled { goto skipDefault }
	if err := orch.processReviewTask(ctx, logger, task, agentmeta.StageApply, orch.Config.NeedCodeReviewStateID); err != nil { return err }
case orch.Config.NeedArchiveAIReviewStateID:
	if !orch.Config.AIReviewEnabled { goto skipDefault }
	if err := orch.processReviewTask(ctx, logger, task, agentmeta.StageArchive, orch.Config.NeedArchiveReviewStateID); err != nil { return err }
skipDefault:
```

(Or refactor without `goto`: extract a small helper that checks AIReviewEnabled inside `processReviewTask` and returns nil + skip log if disabled. Simpler — let's do that:)

Replace the three cases with:

```go
case orch.Config.NeedProposalAIReviewStateID:
	if err := orch.routeReview(ctx, logger, task, agentmeta.StageProposal, orch.Config.NeedProposalReviewStateID); err != nil { return err }
case orch.Config.NeedCodeAIReviewStateID:
	if err := orch.routeReview(ctx, logger, task, agentmeta.StageApply, orch.Config.NeedCodeReviewStateID); err != nil { return err }
case orch.Config.NeedArchiveAIReviewStateID:
	if err := orch.routeReview(ctx, logger, task, agentmeta.StageArchive, orch.Config.NeedArchiveReviewStateID); err != nil { return err }
```

```go
func (orch *Orchestrator) routeReview(ctx context.Context, logger steplog.Logger, task taskmanager.Task, stage agentmeta.Stage, targetState string) error {
	if !orch.Config.AIReviewEnabled {
		logger.Infof(module, "skip ai-review task=%s identifier=%s state=%s reason=feature_disabled", task.ID, task.Identifier, task.State.ID)
		return nil
	}
	if orch.ReviewRunner == nil {
		return fmt.Errorf("ai-review enabled but ReviewRunner is nil")
	}
	return orch.processReviewTask(ctx, logger, task, stage, targetState)
}
```

- [ ] **Step 5: Update validate()**

```go
// internal/coreorch/orchestrator.go validate() — only require ReviewRunner when AIReviewEnabled
if orch.Config.AIReviewEnabled && orch.ReviewRunner == nil {
	return fmt.Errorf("ai review enabled but review runner is nil")
}
if orch.Config.AIReviewEnabled {
	if strings.TrimSpace(orch.Config.NeedProposalAIReviewStateID) == "" {
		return fmt.Errorf("need-proposal-ai-review state id must not be empty when ai review enabled")
	}
	if strings.TrimSpace(orch.Config.NeedCodeAIReviewStateID) == "" {
		return fmt.Errorf("need-code-ai-review state id must not be empty when ai review enabled")
	}
	if strings.TrimSpace(orch.Config.NeedArchiveAIReviewStateID) == "" {
		return fmt.Errorf("need-archive-ai-review state id must not be empty when ai review enabled")
	}
}
```

- [ ] **Step 6: Modify processProposalTask / processApplyTask / processArchiveTask to choose target state by feature flag**

In `processProposalTask`: replace target of final `MoveTask` to:

```go
target := orch.Config.NeedProposalReviewStateID
if orch.Config.AIReviewEnabled { target = orch.Config.NeedProposalAIReviewStateID }
if err := orch.TaskManager.MoveTask(ctx, task.ID, target); err != nil { ... }
```

Same in `processApplyTask` and `processArchiveTask` with their respective AI-review state IDs.

- [ ] **Step 7: Write/extend orchestrator_test.go**

Add tests:

```go
// internal/coreorch/orchestrator_test.go

func TestRunRoutesProposalAIReviewToReviewRunnerWhenEnabled(t *testing.T) {
	taskMgr := &fakeTaskManager{tasks: []taskmanager.Task{{
		ID: "1", Identifier: "ZIM-1", Title: "T", State: taskmanager.WorkflowState{ID: "p-ai"},
		PullRequests: []taskmanager.PullRequest{{URL: "https://github.com/o/r/pull/42", Branch: "b"}},
	}}}
	rev := &fakeReviewRunner{}
	orch := &Orchestrator{
		Config: Config{
			AIReviewEnabled: true,
			ReadyToProposeStateID: "rp", ProposingInProgressStateID: "pip", NeedProposalReviewStateID: "npr",
			ReadyToCodeStateID: "rc", CodeInProgressStateID: "cip", NeedCodeReviewStateID: "ncr",
			ReadyToArchiveStateID: "ra", ArchivingInProgressStateID: "aip", NeedArchiveReviewStateID: "nar",
			NeedProposalAIReviewStateID: "p-ai", NeedCodeAIReviewStateID: "c-ai", NeedArchiveAIReviewStateID: "a-ai",
		},
		TaskManager: taskMgr,
		ProposalRunner: noopProposalRunner{}, ApplyRunner: noopApplyRunner{}, ArchiveRunner: noopArchiveRunner{},
		ReviewRunner: rev, Service: "test",
	}
	if err := orch.RunProposalsOnce(context.Background()); err != nil { t.Fatalf("RunProposalsOnce: %v", err) }
	if rev.calls != 1 || rev.lastInput.Stage != agentmeta.StageProposal {
		t.Fatalf("review calls=%d lastInput=%+v", rev.calls, rev.lastInput)
	}
	if got := taskMgr.lastMove("1"); got != "npr" {
		t.Fatalf("expected move to npr, got %q", got)
	}
}

func TestRunDoesNotRouteAIReviewWhenDisabled(t *testing.T) {
	taskMgr := &fakeTaskManager{tasks: []taskmanager.Task{{
		ID: "1", State: taskmanager.WorkflowState{ID: "p-ai"},
	}}}
	rev := &fakeReviewRunner{}
	orch := &Orchestrator{
		Config: Config{
			AIReviewEnabled: false,
			ReadyToProposeStateID: "rp", ProposingInProgressStateID: "pip", NeedProposalReviewStateID: "npr",
			ReadyToCodeStateID: "rc", CodeInProgressStateID: "cip", NeedCodeReviewStateID: "ncr",
			ReadyToArchiveStateID: "ra", ArchivingInProgressStateID: "aip", NeedArchiveReviewStateID: "nar",
			NeedProposalAIReviewStateID: "p-ai",
		},
		TaskManager: taskMgr,
		ProposalRunner: noopProposalRunner{}, ApplyRunner: noopApplyRunner{}, ArchiveRunner: noopArchiveRunner{},
		Service: "test",
	}
	if err := orch.RunProposalsOnce(context.Background()); err != nil { t.Fatalf("RunProposalsOnce: %v", err) }
	if rev.calls != 0 { t.Fatalf("expected no review calls, got %d", rev.calls) }
}

func TestProposalRouteMovesToAIReviewStateWhenEnabled(t *testing.T) {
	taskMgr := &fakeTaskManager{tasks: []taskmanager.Task{{
		ID: "1", State: taskmanager.WorkflowState{ID: "rp"},
	}}}
	prop := &capturingProposalRunner{prURL: "https://github.com/o/r/pull/42"}
	orch := &Orchestrator{
		Config: Config{
			AIReviewEnabled: true,
			ReadyToProposeStateID: "rp", ProposingInProgressStateID: "pip", NeedProposalReviewStateID: "npr",
			ReadyToCodeStateID: "rc", CodeInProgressStateID: "cip", NeedCodeReviewStateID: "ncr",
			ReadyToArchiveStateID: "ra", ArchivingInProgressStateID: "aip", NeedArchiveReviewStateID: "nar",
			NeedProposalAIReviewStateID: "p-ai", NeedCodeAIReviewStateID: "c-ai", NeedArchiveAIReviewStateID: "a-ai",
		},
		TaskManager: taskMgr, ProposalRunner: prop,
		ApplyRunner: noopApplyRunner{}, ArchiveRunner: noopArchiveRunner{},
		ReviewRunner: &fakeReviewRunner{},
		Service: "test",
	}
	if err := orch.RunProposalsOnce(context.Background()); err != nil { t.Fatalf("RunProposalsOnce: %v", err) }
	if got := taskMgr.lastMove("1"); got != "p-ai" {
		t.Fatalf("expected move to p-ai, got %q", got)
	}
}
```

Add fakes (`fakeReviewRunner`, `noopProposalRunner` etc.) at the bottom of the file. Use existing fake patterns from this test file as reference.

- [ ] **Step 8: Run tests**

Run: `go test ./internal/coreorch/...`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/coreorch/
git commit -m "Route AI review tasks through CoreOrch with feature flag"
```

---

## Task 16: cmd/orchv3 wiring — executor factory and ReviewRunner construction

**Files:**
- Modify: `cmd/orchv3/main.go:21-179`
- Modify: `cmd/orchv3/main_test.go`

Цель: запровайдить `ReviewRunner` со словарём executor'ов по слотам и пробросить producer-trailer-данные в три producer-runner'а.

- [ ] **Step 1: Add singleReviewRunner interface**

```go
type singleReviewRunner interface {
	Run(ctx context.Context, input reviewrunner.ReviewInput) (reviewrunner.Result, error)
}
```

Add `"orchv3/internal/reviewrunner"`, `"orchv3/internal/reviewrunner/prcommenter"`, `"orchv3/internal/agentmeta"` imports.

- [ ] **Step 2: Extend appDeps with newReviewRunner**

```go
type appDeps struct {
	// ... existing
	newReviewRunner func(cfg config.Config, logOut io.Writer) singleReviewRunner
	newProposalOrchestrator func(cfg config.Config, tasks coreorch.TaskManager, proposalRunner coreorch.ProposalRunner, applyRunner coreorch.ApplyRunner, archiveRunner coreorch.ArchiveRunner, reviewRunner coreorch.ReviewRunner, logOut io.Writer) proposalMonitor
}
```

- [ ] **Step 3: Update default factories — set Producer on three runners**

```go
newProposalRunner: func(cfg config.ProposalRunnerConfig, service string, logOut io.Writer) singleProposalRunner {
	runner := proposalrunner.New(cfg)
	runner.Service = service
	runner.Stdout = logOut; runner.Stderr = logOut
	runner.Command = commandrunner.ExecRunner{LogWriter: logOut}
	// Producer wired by caller (defaultDeps doesn't know cfg.Review yet) — see runWithDeps below.
	return runner
},
```

Better: make `runWithDeps` pass full `cfg config.Config` to runner factories. Refactor signature:

```go
newProposalRunner func(cfg config.Config, service string, logOut io.Writer) singleProposalRunner
```

Inside the factory, set `runner.Producer = agentmeta.Producer{By: cfg.Review.PrimarySlot, Model: cfg.Review.PrimaryModel, Stage: agentmeta.StageProposal}` if `cfg.Review.PrimarySlot != ""`.

Same for apply and archive (with their Stage values).

- [ ] **Step 4: Implement defaultDeps().newReviewRunner**

```go
newReviewRunner: func(cfg config.Config, logOut io.Writer) singleReviewRunner {
	if !cfg.Review.Enabled(cfg.TaskManager) {
		return nil
	}
	// Build executors map. Today both slots resolve to Codex CLI with different models.
	executors := map[string]reviewrunner.AgentExecutor{}
	executors[cfg.Review.PrimarySlot] = reviewrunner.CodexCLIExecutor{
		Command: commandrunner.ExecRunner{LogWriter: logOut},
		CodexPath: cfg.Review.PrimaryExecutorPath, Model: cfg.Review.PrimaryModel, Service: cfg.AppName,
	}
	executors[cfg.Review.SecondarySlot] = reviewrunner.CodexCLIExecutor{
		Command: commandrunner.ExecRunner{LogWriter: logOut},
		CodexPath: cfg.Review.SecondaryExecutorPath, Model: cfg.Review.SecondaryModel, Service: cfg.AppName,
	}

	commenter := prcommenter.GHPostReviewCommenter{
		Command: commandrunner.ExecRunner{LogWriter: logOut},
		GHPath: cfg.ProposalRunner.GHPath, Service: cfg.AppName,
	}

	return &reviewrunner.Runner{
		Config: cfg.Review, ProposalCfg: cfg.ProposalRunner,
		Command: commandrunner.ExecRunner{LogWriter: logOut},
		Executors: executors, Commenter: commenter,
		Service: cfg.AppName, Stdout: logOut, Stderr: logOut,
	}
},
```

- [ ] **Step 5: Update newProposalOrchestrator factory**

```go
newProposalOrchestrator: func(cfg config.Config, tasks coreorch.TaskManager, proposalRunner coreorch.ProposalRunner, applyRunner coreorch.ApplyRunner, archiveRunner coreorch.ArchiveRunner, reviewRunner coreorch.ReviewRunner, logOut io.Writer) proposalMonitor {
	return &coreorch.Orchestrator{
		Config: coreorch.Config{
			ReadyToProposeStateID:        cfg.TaskManager.ReadyToProposeStateID,
			ProposingInProgressStateID:   cfg.TaskManager.ProposingInProgressStateID,
			NeedProposalReviewStateID:    cfg.TaskManager.NeedProposalReviewStateID,
			NeedProposalAIReviewStateID:  cfg.TaskManager.NeedProposalAIReviewStateID,
			ReadyToCodeStateID:           cfg.TaskManager.ReadyToCodeStateID,
			CodeInProgressStateID:        cfg.TaskManager.CodeInProgressStateID,
			NeedCodeReviewStateID:        cfg.TaskManager.NeedCodeReviewStateID,
			NeedCodeAIReviewStateID:      cfg.TaskManager.NeedCodeAIReviewStateID,
			ReadyToArchiveStateID:        cfg.TaskManager.ReadyToArchiveStateID,
			ArchivingInProgressStateID:   cfg.TaskManager.ArchivingInProgressStateID,
			NeedArchiveReviewStateID:     cfg.TaskManager.NeedArchiveReviewStateID,
			NeedArchiveAIReviewStateID:   cfg.TaskManager.NeedArchiveAIReviewStateID,
			AIReviewEnabled:              cfg.Review.Enabled(cfg.TaskManager),
		},
		TaskManager: tasks, ProposalRunner: proposalRunner,
		ApplyRunner: applyRunner, ArchiveRunner: archiveRunner,
		ReviewRunner: reviewRunner,
		Service: cfg.AppName, LogWriter: logOut,
	}
},
```

- [ ] **Step 6: Update runWithDeps to construct and pass ReviewRunner**

Inside `runWithDeps`:

```go
reviewRunner := deps.newReviewRunner(cfg, logOut)
var coreReview coreorch.ReviewRunner
if reviewRunner != nil {
	coreReview = reviewRunner
}
orchestrator := deps.newProposalOrchestrator(cfg, taskManager, proposalRunner, applyRunner, archiveRunner, coreReview, logOut)
```

- [ ] **Step 7: Update existing main_test.go expectations**

Read existing main_test.go to find tests that construct `appDeps`. Inject a stub `newReviewRunner` (returns nil) so existing flows keep passing. Add a new test asserting that when AI review is enabled, `coreReview` is non-nil and propagates into orchestrator config.

- [ ] **Step 8: Run all tests**

Run: `go test ./... && go fmt ./...`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add cmd/orchv3/
git commit -m "Wire review runner and producer trailers into orchv3 cli"
```

---

## Task 17: Documentation and .env.example

**Files:**
- Modify: `architecture.md`
- Modify: `docs/proposal-runner.md`
- Create or modify: `.env.example`

- [ ] **Step 1: Append AI-review block to .env.example**

If `.env.example` doesn't exist at repo root, create it. Append at the end:

```
# Cross-Agent Review Stage. All three Linear states must be set together to
# enable AI review; leave them empty to disable the stage entirely.
LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID=
LINEAR_STATE_NEED_CODE_AI_REVIEW_ID=
LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID=

# Reviewer slots. Both slots must be configured when AI review is enabled.
# Today both may point to Codex; in the future REVIEW_ROLE_SECONDARY may
# become claude or another agent. Models must differ for the cross-review to
# make sense.
REVIEW_ROLE_PRIMARY=
REVIEW_ROLE_SECONDARY=
REVIEW_PRIMARY_MODEL=
REVIEW_SECONDARY_MODEL=
REVIEW_PRIMARY_EXECUTOR_PATH=
REVIEW_SECONDARY_EXECUTOR_PATH=

# Optional review runtime knobs.
REVIEW_MAX_CONTEXT_BYTES=
REVIEW_PARSE_REPAIR_RETRIES=
REVIEW_PROMPT_DIR=
```

- [ ] **Step 2: Add Review-Stage flow to architecture.md**

Insert a new section between «Целевой Поток Archive-Stage» and «Границы Ответственности»:

```markdown
## Целевой Поток Review-Stage

1. `CoreOrch` в том же проходе monitor-а получает managed tasks от `TaskManager`.
2. Задачи в `Need * AI Review` (Proposal/Code/Archive) маршрутизируются в Review-stage соответствующего этапа.
3. `ReviewRunner` клонирует ветку задачи во временную директорию, читает producer-trailer последнего HEAD-коммита и выбирает reviewer-слот, противоположный продьюсеру.
4. Reviewer-executor запускается с stage-specific prompt'ом, возвращает строго JSON по схеме review-ответа; при невалидном JSON выполняется один repair-retry.
5. Распарсенный review публикуется одним атомарным POST'ом через GitHub Pull Request Reviews API: summary в body PR review плюс inline-комментарии на каждую находку с собственным fix-prompt'ом.
6. Идемпотентность по HTML-маркеру `(reviewer, stage, HEAD-sha)`: повторный запуск review на том же коммите пропускает публикацию и сразу переходит к смене статуса.
7. После успешной публикации (или idempotent skip) `CoreOrch` переводит задачу в человеческий review-state соответствующей стадии.
8. При сбое публикации, невалидном JSON после repair или config-mismatch reviewer-слота задача остаётся в AI-review state, monitor подхватит её следующим тиком.
```

In «Маппинг На Текущий Код» append:

```markdown
- `ReviewRunner` реализован в `internal/reviewrunner` как четвёртая stage-агностичная реализация над `AgentExecutor`-контрактом. Инкапсулирует строгий JSON-парсер review-ответа (`internal/reviewrunner/reviewparse`), формирование PR review (`internal/reviewrunner/prcommenter`) и stage-specific prompt-templates (`internal/reviewrunner/prompts/*.tmpl`).
- `internal/agentmeta` хранит контракт producer-trailer'а в commit message: `Produced-By`, `Produced-Model`, `Produced-Stage`. Используется тремя producer-runner'ами при коммите и `ReviewRunner` при чтении HEAD.
```

- [ ] **Step 3: Add a paragraph to docs/proposal-runner.md**

Insert near the «Запуск» description (after the existing "После создания pull request..." paragraph):

```markdown
Если включена `Cross-Agent Review`-стадия (см. `LINEAR_STATE_NEED_*_AI_REVIEW_ID` и `REVIEW_ROLE_*` переменные), proposal-runner после push переводит задачу не сразу в `Need Proposal Review`, а в `Need Proposal AI Review`. Из этого state monitor подхватывает задачу следующим тиком и запускает `ReviewRunner`, который публикует автоматическое PR review «противоположной» моделью с inline-комментариями и fix-prompt'ами; только после публикации задача переходит к человеку. Apply и Archive стадии работают по тому же паттерну.
```

- [ ] **Step 4: Verify build and tests still pass**

Run: `go fmt ./... && go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add .env.example architecture.md docs/proposal-runner.md
git commit -m "Document cross-agent review stage and configuration"
```

---

## Self-Review

**Spec coverage check:**

| Spec section | Plan coverage |
|---|---|
| Архитектура и роли | Tasks 14-15 (ReviewRunner + CoreOrch routes) |
| State machine + Linear states | Tasks 5, 15 |
| Producer marker (commit trailer) | Tasks 1-4 |
| Reviewer slot selection | Tasks 5 (config), 12 (selection) |
| ReviewRunner contract + JSON | Tasks 7, 14 |
| Categories per stage | Task 8 |
| Severity / verdict | Task 8 |
| Prompt templates | Task 10 |
| fix_prompt rules | Task 10 (template body) |
| PR comments via gh api | Task 11 |
| Idempotency by HEAD-sha | Task 11 |
| Feature flag | Tasks 5, 15-16 |
| Testing strategy | Each task includes table-driven tests |
| Logging (review module) | Task 14 (logger.Infof calls) |
| .env.example, architecture, docs | Task 17 |
| Migration plan | Tasks 1→17 follow the spec's migration order |
| Rollback (clear AI-review state IDs) | Tasks 5, 15 (feature-flag) |

No spec section without a task. ✅

**Placeholder scan:** No `TBD`/`TODO`/«fill in details» in steps. Each step shows actual code or commands. ✅

**Type consistency check:**

- `agentmeta.Producer{By, Model, Stage}` — same fields used in Tasks 1, 2, 3, 4, 12, 14, 16. ✅
- `agentmeta.Stage` constants `StageProposal/StageApply/StageArchive` — used identically across tasks. ✅
- `reviewparse.Review/Summary/Finding` — same struct shape in Tasks 7, 8, 11, 14. ✅
- `prcommenter.PostReviewInput` fields — same in Task 11 (definition) and Task 14 (caller). ✅
- `reviewrunner.AgentExecutor` interface — same in Task 13 (definition) and Task 14 (use). ✅
- `coreorch.Config.AIReviewEnabled bool` — same in Tasks 15 (definition) and 16 (set in wiring). ✅
- `coreorch.ReviewRunner` interface — defined in Task 15, satisfied by `*reviewrunner.Runner` (Task 14). ✅

No naming drift detected.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-28-cross-agent-review.md`. Two execution options:

1. **Subagent-Driven (recommended)** — я диспатчу свежего сабагента на каждую задачу с двухступенчатым review между ними. Быстрее итерируемся, меньше дрифта контекста.
2. **Inline Execution** — выполняю задачи прямо в этой сессии через executing-plans с чекпоинтами.

Which approach?
