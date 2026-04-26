## MODIFIED Requirements

### Requirement: Codex CLI openspec propose execution
The system SHALL execute the OpenSpec proposal generation step through an internal `AgentExecutor` contract. The default implementation SHALL remain Codex CLI and SHALL preserve the current local non-interactive command format `codex exec --json --sandbox danger-full-access --output-last-message <path> --cd <clone-dir> -`, with the prompt passed through stdin and containing the `openspec-propose` skill instruction plus the original task description.

#### Scenario: Agent executor receives proposal task
- **WHEN** the workflow reaches the agent proposal step
- **THEN** the proposal runner invokes its configured `AgentExecutor` with the original task description and the clone workspace path

#### Scenario: Codex executor receives prompt
- **WHEN** the configured `AgentExecutor` is the default Codex CLI implementation
- **THEN** the Codex executor logs the prompt and invokes `codex exec --json --sandbox danger-full-access --output-last-message <path> --cd <clone-dir> -` with that prompt on stdin

#### Scenario: Agent executor succeeds
- **WHEN** the `AgentExecutor` exits successfully after creating OpenSpec artifacts
- **THEN** the system continues to git status, commit, push, and PR creation

#### Scenario: Agent executor fails
- **WHEN** the `AgentExecutor` returns an error
- **THEN** the system logs agent output and returns an error that identifies the agent proposal step

### Requirement: Console logging of workflow steps
The system SHALL log all workflow steps and agent execution interaction to the console as JSON Lines application log events, including prompt text, command output, PR creation progress, and final PR URL.

#### Scenario: Workflow emits structured step logs
- **WHEN** the proposal runner executes a workflow
- **THEN** the console output includes JSON log events for temp directory creation, git clone, agent prompt or execution start, agent output, git commit/push, PR creation, and final PR URL

#### Scenario: Workflow logs include required fields
- **WHEN** the proposal runner writes a workflow log event
- **THEN** the event contains `time`, `module`, `type`, and `message` fields

#### Scenario: Agent emits reasoning or output
- **WHEN** the configured agent runtime writes reasoning, progress, stderr, or final output to its process streams
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

### Requirement: Pull request creation
The system SHALL create a pull request through the authenticated `gh` CLI in the target GitHub repository after the agent executor produces changes and SHALL return the pull request URL.

#### Scenario: Pull request is created
- **WHEN** the agent executor succeeds and the cloned repository has changes to commit
- **THEN** the system commits the changes, pushes a branch, creates a PR through `gh`, logs the PR URL, and returns that URL to the caller

#### Scenario: GitHub CLI is unavailable or unauthenticated
- **WHEN** PR creation requires `gh` but `gh` is missing or not authenticated
- **THEN** the system returns an error that identifies the GitHub CLI prerequisite

#### Scenario: No changes were produced
- **WHEN** the agent executor succeeds but git status shows no changes
- **THEN** the system returns an error and does not create an empty pull request

#### Scenario: PR creation fails
- **WHEN** the PR creation command exits with an error
- **THEN** the system logs the PR creation output and returns an error that identifies the PR step

### Requirement: Testable command execution
The proposal runner module SHALL allow tests to replace agent execution and external command execution so unit tests do not require real GitHub access, Codex CLI, or network calls.

#### Scenario: Agent executor is substituted in tests
- **WHEN** a unit test constructs the proposal runner with a fake agent executor
- **THEN** the test can assert the workflow around clone, git status, branch, commit, push, PR creation, and PR comment without executing a real agent runtime

#### Scenario: Command runner is substituted in Codex executor tests
- **WHEN** a unit test constructs the Codex CLI executor with a fake command runner
- **THEN** the test can assert the Codex command, arguments, stdin prompt, output forwarding, and last-message capture without executing Codex CLI

### Requirement: Codex final response PR comment
The system SHALL publish the last non-empty agent response as a separate comment on the created pull request. The default Codex CLI implementation SHALL obtain that response from `codex exec --output-last-message`.

#### Scenario: Final agent response is present
- **WHEN** the workflow creates a pull request and the agent executor returns a non-empty final message
- **THEN** the system publishes that message as a pull request comment and logs the comment creation step

#### Scenario: Final agent response is empty
- **WHEN** the workflow creates a pull request and the agent executor returns an empty or whitespace-only final message
- **THEN** the system does not create an empty pull request comment and still returns the pull request URL

#### Scenario: Agent response comment fails
- **WHEN** the pull request is created but publishing the last agent message as a comment fails
- **THEN** the system returns an error that identifies the comment step and logs the comment creation output
