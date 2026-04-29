## ADDED Requirements

### Requirement: Proposal runner contract survives runner module relocation

The proposal runner SHALL preserve its existing input validation, Codex CLI invocation, GitManager delegation, pull request creation, final-response comment, structured logging, and test substitution behavior after being moved under the dedicated runner module area and after adopting shared runner components.

#### Scenario: Proposal input validation is unchanged

- **WHEN** the proposal runner receives a `ProposalInput` with empty title or empty agent prompt
- **THEN** it returns a validation error before creating a temp workspace or running external commands

#### Scenario: Proposal Codex command is unchanged

- **WHEN** the default proposal agent executor invokes Codex CLI
- **THEN** it uses the same non-interactive `codex exec --json --sandbox danger-full-access --output-last-message <path> --cd <clone-dir> -` command shape
- **AND** it passes the proposal prompt through stdin

#### Scenario: Proposal PR workflow is unchanged

- **WHEN** the proposal agent succeeds and the clone workspace has changes
- **THEN** the proposal runner delegates clone status, branch checkout, commit, push, pull request creation, and optional final-message comment to `GitManager`
- **AND** it returns the created pull request URL

#### Scenario: Proposal no-change behavior is unchanged

- **WHEN** the proposal agent succeeds but repository status is empty
- **THEN** the proposal runner returns the existing no-changes error
- **AND** it does not create an empty commit or pull request

#### Scenario: Proposal runner remains testable

- **WHEN** tests construct the relocated proposal runner with fake agent, command, or git dependencies
- **THEN** they can assert the proposal workflow without real GitHub access, Codex CLI, Git commands, or network calls
