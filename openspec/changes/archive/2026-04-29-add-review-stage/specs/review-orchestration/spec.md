## ADDED Requirements

### Requirement: Review orchestration uses AI-review tasks across all three stages
The system SHALL provide a Review orchestration stage that loads managed tasks through `TaskManager` and processes tasks whose workflow state ID matches one of `LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID`, `LINEAR_STATE_NEED_CODE_AI_REVIEW_ID`, or `LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID`. Each AI-review state SHALL route to the Review runner with an explicit Stage value (`Proposal`, `Apply`, `Archive`) so stage-specific prompts, categories, and targets can be selected.

#### Scenario: Proposal AI-review task is selected
- **WHEN** `TaskManager` returns a task whose state ID equals `LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID`
- **THEN** the orchestration stage calls the Review runner with Stage = Proposal

#### Scenario: Apply AI-review task is selected
- **WHEN** `TaskManager` returns a task whose state ID equals `LINEAR_STATE_NEED_CODE_AI_REVIEW_ID`
- **THEN** the orchestration stage calls the Review runner with Stage = Apply

#### Scenario: Archive AI-review task is selected
- **WHEN** `TaskManager` returns a task whose state ID equals `LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID`
- **THEN** the orchestration stage calls the Review runner with Stage = Archive

#### Scenario: Non-review managed task is skipped by Review route
- **WHEN** `TaskManager` returns a task from another managed state such as ready-to-propose, ready-to-code, ready-to-archive, code-in-progress, or archiving-in-progress
- **THEN** the Review route does not call the Review runner for that task

### Requirement: Review feature-flag is controlled by AI-review state IDs and reviewer slots
The system SHALL treat the Review orchestration stage as enabled only when all three AI-review state IDs and both reviewer slot configurations are non-empty. When all three AI-review state IDs are empty, the Review route SHALL not be registered and producer runners SHALL fall back to transitioning tasks directly to the human review state of their stage. Partial configuration SHALL fail orchestration startup with a contextual error.

#### Scenario: All AI-review state IDs and reviewer slots configured
- **WHEN** all three AI-review state IDs and both reviewer slot configurations are configured
- **THEN** the orchestration stage registers the Review route
- **AND** producer runners transition tasks to the AI-review state of their stage

#### Scenario: All AI-review state IDs empty
- **WHEN** all three AI-review state IDs are empty
- **THEN** the orchestration stage does not register the Review route
- **AND** producer runners transition tasks directly to the human review state of their stage

#### Scenario: Partial AI-review configuration is rejected
- **WHEN** at least one AI-review state ID is configured but not all of them, or reviewer slots are configured inconsistently with the AI-review state IDs
- **THEN** orchestration startup returns a contextual configuration error
- **AND** the orchestrator does not start

### Requirement: Producer marker is recorded as a git trailer in the commit message
The system SHALL append a producer marker as canonical git trailers to the commit message of every artifact produced by `ProposalRunner`, `ApplyRunner`, and `ArchiveRunner`. The trailer SHALL include `Produced-By` (slot identifier such as `codex` or `claude`), `Produced-Model` (concrete model identifier), and `Produced-Stage` (`proposal`, `apply`, or `archive`). The trailer format SHALL be readable by `git interpret-trailers --parse` and SHALL be parsed case-insensitively on the key.

#### Scenario: Proposal commit carries producer trailer
- **WHEN** the proposal runner commits agent-produced changes with AI review enabled
- **THEN** the commit message contains `Produced-By`, `Produced-Model`, and `Produced-Stage: proposal` trailers

#### Scenario: Apply commit carries producer trailer
- **WHEN** the apply runner commits agent-produced changes with AI review enabled
- **THEN** the commit message contains `Produced-By`, `Produced-Model`, and `Produced-Stage: apply` trailers

#### Scenario: Archive commit carries producer trailer
- **WHEN** the archive runner commits agent-produced changes with AI review enabled
- **THEN** the commit message contains `Produced-By`, `Produced-Model`, and `Produced-Stage: archive` trailers

### Requirement: Review runner selects reviewer slot opposite to the latest producer
The system SHALL parse the producer trailer of the most recent HEAD commit on the task branch and select the reviewer slot as the opposite of the producer slot. When the producer slot equals the configured `REVIEW_ROLE_PRIMARY`, the reviewer SHALL be the slot configured as `REVIEW_ROLE_SECONDARY`, and vice versa. Earlier commits in the branch history SHALL NOT influence reviewer selection.

#### Scenario: Producer is primary slot
- **WHEN** the most recent HEAD commit on the task branch carries `Produced-By: <REVIEW_ROLE_PRIMARY value>`
- **THEN** the Review runner uses the executor and model configured for `REVIEW_ROLE_SECONDARY`

#### Scenario: Producer is secondary slot
- **WHEN** the most recent HEAD commit on the task branch carries `Produced-By: <REVIEW_ROLE_SECONDARY value>`
- **THEN** the Review runner uses the executor and model configured for `REVIEW_ROLE_PRIMARY`

#### Scenario: Producer trailer is missing
- **WHEN** the most recent HEAD commit on the task branch carries no producer trailer
- **THEN** the Review runner uses the executor and model configured for `REVIEW_ROLE_SECONDARY`
- **AND** the Review runner emits a warning log event identifying the absent trailer
- **AND** the published review summary includes a "producer unknown" tripwire

#### Scenario: Producer trailer references unknown slot
- **WHEN** the producer trailer specifies a slot identifier that matches neither configured slot
- **THEN** the Review runner returns a contextual configuration error
- **AND** the Review runner does not call any executor
- **AND** the orchestration stage does not move the task to the human review state

### Requirement: Reviewer returns strict JSON matching the review schema
The system SHALL prompt the reviewer executor with a strict JSON output contract and SHALL parse the response as JSON conforming to the review schema. The schema SHALL contain a `summary` object with `verdict` (one of `ship-ready`, `needs-work`, `blocked`), `walkthrough` (markdown string), and `stats` (counts by severity), and a `findings` array where each finding contains `id`, `category` (closed enum scoped to the stage), `severity` (one of `blocker`, `major`, `minor`, `nit`), `file`, `line_start` (integer or null), `line_end` (integer or null), `title`, `message`, and `fix_prompt`. When the first reviewer response fails to parse or validate, the Review runner SHALL retry exactly once with a repair prompt that includes the validation error.

#### Scenario: Valid JSON response is parsed and used
- **WHEN** the reviewer executor returns a JSON response that conforms to the schema
- **THEN** the Review runner uses the parsed summary and findings to build the PR review

#### Scenario: Invalid JSON triggers exactly one repair attempt
- **WHEN** the first reviewer response fails JSON parsing or schema validation
- **THEN** the Review runner sends a repair prompt that includes the validation error
- **AND** the repair prompt asks the executor to return strict JSON conforming to the schema

#### Scenario: Repair attempt also fails
- **WHEN** both the initial response and the repair response fail to parse or validate
- **THEN** the Review runner returns a contextual error
- **AND** the orchestration stage does not move the task to the human review state
- **AND** the orchestration stage does not publish a partial review

### Requirement: Review categories are closed and stage-specific
The system SHALL constrain each finding's `category` to a closed enum scoped to the Review stage. The Proposal stage enum SHALL include `requirement_unclear`, `requirement_contradicts_existing`, `scenario_missing`, `acceptance_criteria_weak`, `scope_creep`, `tasks_misaligned`, `architecture_violation`, and `nit`. The Apply stage enum SHALL include `spec_mismatch`, `bug`, `error_handling`, `concurrency`, `test_gap`, `architecture_violation`, `idiom`, `config_drift`, and `nit`. The Archive stage enum SHALL include `incomplete_archive`, `spec_drift`, `dangling_reference`, `metadata_missing`, and `nit`.

#### Scenario: Reviewer chooses category from the stage enum
- **WHEN** the reviewer returns a finding for a Proposal-stage review
- **THEN** the finding's category is one of the Proposal-stage enum values

#### Scenario: Out-of-enum category fails parsing
- **WHEN** a reviewer response includes a finding with a category not in the stage enum
- **THEN** parsing fails and the runner attempts the configured repair retry

### Requirement: Severity influences presentation but does not gate transitions
The system SHALL treat finding severity as informational. Severity SHALL determine the summary `verdict` value, the order of findings in the summary, and the icon shown next to each inline comment. Severity SHALL NOT block, defer, or change the transition of the task into the human review state. Human reviewers remain solely responsible for accepting or rejecting the artefact.

#### Scenario: Blocker findings do not block transition
- **WHEN** a published review contains at least one finding with severity `blocker`
- **THEN** the Review runner still moves the task to the human review state after publication

#### Scenario: Verdict reflects severity distribution
- **WHEN** the parsed findings contain at least one `blocker`
- **THEN** the summary verdict is `blocked`

#### Scenario: Verdict is needs-work when major findings present
- **WHEN** the parsed findings contain at least one `major` and no `blocker`
- **THEN** the summary verdict is `needs-work`

#### Scenario: Verdict is ship-ready otherwise
- **WHEN** the parsed findings contain only `minor` or `nit` severities or no findings at all
- **THEN** the summary verdict is `ship-ready`

### Requirement: Review runner publishes one atomic PR review with summary and inline comments
The system SHALL publish review output as a single atomic Pull Request review through the GitHub Pull Request Reviews API with `event` set to `COMMENT`. The review body SHALL contain a summary derived from the parsed `summary` object and a list of findings. Each finding with non-null `line_start` SHALL be published as an inline comment attached to the corresponding file and line range on the HEAD commit. Each inline comment SHALL include a model prefix, severity icon, category label, the finding message, and a `<details>` block titled `🤖 Prompt for AI Agent` containing the `fix_prompt` text.

#### Scenario: Single POST publishes summary and all inline comments
- **WHEN** parsed findings include at least one finding with non-null `line_start`
- **THEN** the Review runner sends a single POST to `/repos/{owner}/{repo}/pulls/{number}/reviews`
- **AND** the request body contains the summary and all inline comments
- **AND** the request `event` is `COMMENT`

#### Scenario: General findings appear only in summary
- **WHEN** a finding has `line_start` set to null
- **THEN** the finding appears in the summary findings list
- **AND** the finding is not sent as an inline comment

#### Scenario: Inline comment carries model prefix and fix prompt
- **WHEN** the Review runner formats an inline comment for a finding
- **THEN** the comment body includes a `[review by <reviewer slot>]` prefix
- **AND** the comment body includes the severity icon and category label
- **AND** the comment body includes the finding message
- **AND** the comment body includes a collapsible `🤖 Prompt for AI Agent` section containing the fix_prompt text

### Requirement: PR review publication is idempotent by reviewer, stage, and HEAD sha
The system SHALL embed an HTML comment marker `<!-- drop-forge-review-marker:<reviewer-slot>:<stage>:<HEAD-sha> -->` as the first line of the review body. Before publishing, the Review runner SHALL fetch existing reviews on the pull request and SHALL skip publication when a review with the same marker already exists. When publication is skipped due to an existing marker, the Review runner SHALL still proceed to move the task to the human review state.

#### Scenario: First review for HEAD is published
- **WHEN** the pull request has no existing review with the matching marker
- **THEN** the Review runner publishes a new review

#### Scenario: Repeated review for same HEAD is skipped
- **WHEN** the pull request already contains a review whose body starts with the matching marker
- **THEN** the Review runner does not publish another review
- **AND** the Review runner moves the task to the human review state

#### Scenario: New producer push triggers a fresh review
- **WHEN** the producer pushes a new commit and HEAD sha changes
- **THEN** the existing marker no longer matches
- **AND** the Review runner publishes a new review with the updated marker

### Requirement: Review runner uses an isolated temporary clone
The system SHALL execute Review by cloning the configured repository into a temporary directory, checking out the task branch, reading targets and the producer trailer from the working tree, and discarding the workspace according to cleanup configuration consistent with other stage runners.

#### Scenario: Review runner clones into temporary workspace
- **WHEN** the Review runner starts for a valid AI-review task
- **THEN** it creates a temporary workspace
- **AND** it clones the configured repository into that workspace
- **AND** it checks out the task branch before reading targets

#### Scenario: Review runner respects cleanup configuration
- **WHEN** Review execution completes successfully or with an error
- **THEN** the Review runner preserves or removes the temporary workspace according to the same cleanup configuration as other stage runners

### Requirement: Review runner collects stage-specific targets within a context budget
The system SHALL collect prompt targets according to the Review stage. For Proposal, targets SHALL include all files of the new OpenSpec change directory (`proposal.md`, `design.md`, `tasks.md`, and every file under `specs/`). For Apply, targets SHALL include the diff between the merge base and HEAD plus the corresponding OpenSpec change as context. For Archive, targets SHALL include the diff between the merge base and HEAD plus the archived spec files. When the total target size exceeds `REVIEW_MAX_CONTEXT_BYTES`, the Review runner SHALL truncate by priority and SHALL record a "context truncated" tripwire in the published summary.

#### Scenario: Proposal stage targets include change files
- **WHEN** the Review runner collects targets for a Proposal-stage task
- **THEN** the targets include the change's `proposal.md`, `design.md`, `tasks.md`, and every file under `specs/`

#### Scenario: Apply stage targets include diff and change context
- **WHEN** the Review runner collects targets for an Apply-stage task
- **THEN** the targets include the diff between the merge base and HEAD
- **AND** the targets include the corresponding OpenSpec change files as context

#### Scenario: Archive stage targets include diff and archived specs
- **WHEN** the Review runner collects targets for an Archive-stage task
- **THEN** the targets include the diff between the merge base and HEAD
- **AND** the targets include the archived spec files

#### Scenario: Targets exceed context budget
- **WHEN** the total target size exceeds `REVIEW_MAX_CONTEXT_BYTES`
- **THEN** the Review runner truncates targets by priority
- **AND** the published summary records a "context truncated" tripwire

### Requirement: Successful review moves the task to the human review state
The system SHALL move the task from the AI-review state of its stage to the human review state of the same stage after the Review runner successfully publishes a review or determines that an idempotent review already exists. The Review runner SHALL NOT move the task when publication fails, when JSON parsing fails after the repair attempt, or when configuration errors prevent reviewer selection.

#### Scenario: Proposal AI review succeeds
- **WHEN** the Review runner completes successfully for a task in `LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID`
- **THEN** the orchestration stage moves the task to `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`

#### Scenario: Apply AI review succeeds
- **WHEN** the Review runner completes successfully for a task in `LINEAR_STATE_NEED_CODE_AI_REVIEW_ID`
- **THEN** the orchestration stage moves the task to `LINEAR_STATE_NEED_CODE_REVIEW_ID`

#### Scenario: Archive AI review succeeds
- **WHEN** the Review runner completes successfully for a task in `LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID`
- **THEN** the orchestration stage moves the task to `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID`

#### Scenario: Review failure leaves task in AI-review state
- **WHEN** the Review runner returns an error for any reason
- **THEN** the orchestration stage does not move the task
- **AND** the task remains in its current AI-review state for retry on the next monitor pass

### Requirement: Review runner emits structured logs
The system SHALL log Review orchestration decisions and outcomes through the existing structured logger using the module name `review`. Logged events SHALL include start with stage and identity, clone, target collection, executor execution, parse, publish, idempotency skip, and task transition.

#### Scenario: Successful review logs full lifecycle
- **WHEN** the Review runner completes successfully
- **THEN** the logs include structured events for clone, read targets, execute, parse, publish, and move task

#### Scenario: Idempotent skip is logged
- **WHEN** the Review runner finds a matching marker and skips publication
- **THEN** the logs include a structured event identifying the skipped publication

#### Scenario: Parse failure is logged
- **WHEN** JSON parsing fails after the repair attempt
- **THEN** the logs include a structured error event identifying the parse failure

### Requirement: Review dependencies support tests without external systems
The Review orchestration stage SHALL allow tests to replace the agent executor, the PR commenter, the task manager, and the command runner without network access, GitHub CLI, git, or Codex CLI calls.

#### Scenario: Reviewer executor is substituted in tests
- **WHEN** a unit test constructs the Review runner with a fake agent executor
- **THEN** the test asserts review behaviour without Codex CLI or Claude CLI calls

#### Scenario: PR commenter is substituted in tests
- **WHEN** a unit test constructs the Review runner with a fake PR commenter
- **THEN** the test asserts publication and idempotency behaviour without GitHub CLI calls

#### Scenario: Task manager is substituted in tests
- **WHEN** a unit test constructs orchestration with a fake task manager
- **THEN** the test asserts AI-review filtering and state transitions without Linear API calls
