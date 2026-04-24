## MODIFIED Requirements

### Requirement: Runtime configuration from environment files
The system SHALL read runtime configuration from `.env` with `github.com/joho/godotenv` and environment variables, including the target GitHub repository, branch settings, selected proposal agent CLI, and external command paths.

#### Scenario: Repository is configured in env
- **WHEN** `.env` or the process environment contains the target repository setting
- **THEN** the system uses that repository for `git clone`

#### Scenario: Required repository setting is missing
- **WHEN** the target repository setting is absent
- **THEN** the system returns a configuration error before running the proposal workflow

#### Scenario: Environment overrides dot env
- **WHEN** the same configuration key exists in `.env` and in the process environment
- **THEN** the system uses the process environment value

#### Scenario: Dot env syntax is parsed by godotenv
- **WHEN** `.env` contains values that rely on supported godotenv syntax such as quoted strings or inline comments
- **THEN** the system loads those values using godotenv-compatible parsing

#### Scenario: Codex remains the default proposal agent
- **WHEN** the selected proposal agent CLI setting is absent
- **THEN** the system uses Codex CLI for the agent step

#### Scenario: Claude proposal agent is configured
- **WHEN** `.env` or the process environment selects Claude as the proposal agent CLI
- **THEN** the system uses the configured Claude CLI path for the agent step

#### Scenario: Unknown proposal agent is configured
- **WHEN** `.env` or the process environment selects an unsupported proposal agent CLI
- **THEN** the system returns a configuration error before creating a temp directory or running external commands

#### Scenario: Active agent path is missing
- **WHEN** the selected proposal agent CLI has an empty executable path
- **THEN** the system returns a configuration error that identifies the missing path for the selected agent

### Requirement: Environment variable template
The repository SHALL keep `.env.example` synchronized with all supported configuration keys without secrets or default values.

#### Scenario: Template lists runtime keys
- **WHEN** a developer needs to configure the proposal runner locally
- **THEN** `.env.example` lists the required keys without committed values

#### Scenario: Template lists agent selection keys
- **WHEN** a developer needs to choose between Codex CLI and Claude CLI
- **THEN** `.env.example` lists the proposal agent selection key and the supported agent executable path keys without committed values

### Requirement: Codex CLI openspec propose execution
The system SHALL run the selected proposal agent CLI in the cloned repository, pass the OpenSpec proposal prompt through stdin, and preserve the current Codex CLI command format when Codex is selected.

#### Scenario: Codex CLI receives prompt
- **WHEN** the workflow reaches the agent step with Codex selected
- **THEN** the system logs the prompt and invokes `codex exec --json --sandbox danger-full-access --output-last-message <path> --cd <clone-dir> -` with that prompt on stdin

#### Scenario: Codex CLI succeeds
- **WHEN** Codex CLI exits successfully after creating OpenSpec artifacts
- **THEN** the system continues to git status, commit, push, and PR creation

#### Scenario: Codex CLI fails
- **WHEN** Codex CLI exits with a non-zero status
- **THEN** the system logs Codex output and returns an error that identifies the Codex step

#### Scenario: Claude CLI receives prompt
- **WHEN** the workflow reaches the agent step with Claude selected
- **THEN** the system logs the prompt and invokes Claude CLI in non-interactive print mode in `<clone-dir>` with that prompt on stdin

#### Scenario: Claude CLI succeeds
- **WHEN** Claude CLI exits successfully after creating OpenSpec artifacts
- **THEN** the system continues to git status, commit, push, and PR creation

#### Scenario: Claude CLI fails
- **WHEN** Claude CLI exits with a non-zero status
- **THEN** the system logs Claude output and returns an error that identifies the Claude step

### Requirement: Console logging of workflow steps
The system SHALL log all workflow steps and selected agent CLI interaction to the console as JSON Lines application log events, including prompt text, command output, PR creation progress, and final PR URL.

#### Scenario: Workflow emits structured step logs
- **WHEN** the proposal runner executes a workflow
- **THEN** the console output includes JSON log events for temp directory creation, git clone, selected agent prompt, selected agent output, git commit/push, PR creation, and final PR URL

#### Scenario: Workflow logs include required fields
- **WHEN** the proposal runner writes a workflow log event
- **THEN** the event contains `time`, `module`, `type`, and `message` fields

#### Scenario: Codex emits reasoning or agent output
- **WHEN** Codex CLI writes reasoning, progress, stderr, or final output to its process streams
- **THEN** the system forwards that output to the console as JSON log events without filtering it out

#### Scenario: Claude emits reasoning or agent output
- **WHEN** Claude CLI writes progress, stderr, JSON output, or final output to its process streams
- **THEN** the system forwards that output to the console as JSON log events without filtering it out

#### Scenario: Workflow failure emits error log
- **WHEN** a workflow step fails after logging has been initialized
- **THEN** the system writes a JSON log event with `type` set to `error` and a `message` that identifies the failed step

#### Scenario: CLI startup emits structured log
- **WHEN** the CLI starts without a proposal task description
- **THEN** the startup message is written as a JSON log event with `module` set to `cli` and `type` set to `info`

#### Scenario: CLI fatal error emits structured log
- **WHEN** the CLI cannot load configuration, read the task description, or run the proposal workflow
- **THEN** the failure is written as a JSON log event with `module` set to `cli` and `type` set to `error` before the process exits

### Requirement: Codex final response PR comment
The system SHALL publish the last non-empty selected agent response as a separate comment on the created pull request.

#### Scenario: Final Codex response is present
- **WHEN** the workflow creates a pull request with Codex selected and `codex exec` produced a non-empty last message
- **THEN** the system publishes that message as a pull request comment and logs the comment creation step

#### Scenario: Final Codex response is empty
- **WHEN** the workflow creates a pull request with Codex selected and the captured last Codex message is empty or whitespace-only
- **THEN** the system does not create an empty pull request comment and still returns the pull request URL

#### Scenario: Final Claude response is present
- **WHEN** the workflow creates a pull request with Claude selected and Claude CLI output contains a non-empty final response
- **THEN** the system publishes that response as a pull request comment and logs the comment creation step

#### Scenario: Final Claude response is empty
- **WHEN** the workflow creates a pull request with Claude selected and the captured final Claude response is empty or whitespace-only
- **THEN** the system does not create an empty pull request comment and still returns the pull request URL

#### Scenario: Agent response comment fails
- **WHEN** the pull request is created but publishing the last selected agent message as a comment fails
- **THEN** the system returns an error that identifies the comment step and logs the comment creation output
