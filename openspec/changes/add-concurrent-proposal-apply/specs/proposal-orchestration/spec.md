## ADDED Requirements

### Requirement: Proposal and Apply orchestration can run concurrently
The system SHALL run proposal and Apply task processing in separate goroutines during one orchestration pass so that one ready-to-propose task and one ready-to-code task can execute at the same time.

#### Scenario: Proposal and Apply start without waiting for each other
- **WHEN** one orchestration pass receives at least one task in `LINEAR_STATE_READY_TO_PROPOSE_ID` and at least one task in `LINEAR_STATE_READY_TO_CODE_ID`
- **THEN** the system starts proposal processing and Apply processing in separate goroutines
- **AND** neither stage waits for the other stage's runner to finish before starting its own eligible task

#### Scenario: Stage lifecycle remains ordered inside each task
- **WHEN** a proposal task and an Apply task are processed concurrently
- **THEN** the proposal task still moves to proposing-in-progress before its proposal runner starts
- **AND** the proposal task still attaches its PR URL and moves to proposal review only after its proposal runner succeeds
- **AND** the Apply task still moves to code-in-progress before its Apply runner starts
- **AND** the Apply task still moves to code review only after its Apply runner succeeds

#### Scenario: Same-stage tasks remain sequential
- **WHEN** one orchestration pass receives multiple ready-to-propose tasks or multiple ready-to-code tasks
- **THEN** the system processes tasks within the same stage one at a time in the order returned by `TaskManager`
- **AND** the system does not run two proposal runners or two Apply runners concurrently for those tasks

#### Scenario: Stage failure does not cancel already running peer stage
- **WHEN** proposal processing fails while Apply processing is already running in the same pass
- **THEN** the system lets the Apply processing finish its current lifecycle
- **AND** the orchestration pass returns an error that includes the proposal failure after the running stage goroutines complete

#### Scenario: Multiple stage failures are reported
- **WHEN** proposal processing and Apply processing both fail in the same pass
- **THEN** the orchestration pass returns an error that preserves context for both failed stages

## MODIFIED Requirements

### Requirement: Existing proposal runner is used as the proposal executor

The system SHALL execute proposals by calling the existing proposal runner contract with prepared `ProposalInput` values and SHALL not change the proposal runner's internal git, Codex, PR, or comment workflow as part of proposal orchestration.

#### Scenario: Proposal runner is called for eligible task

- **WHEN** a ready-to-propose task is processed
- **THEN** the orchestration stage calls the proposal runner with the prepared `ProposalInput`

#### Scenario: Multiple proposal tasks are processed sequentially

- **WHEN** multiple ready-to-propose tasks are returned in one orchestration pass
- **THEN** the orchestration stage runs the proposal runner for one proposal task at a time in the returned order

#### Scenario: Proposal runner may overlap with Apply runner

- **WHEN** one orchestration pass receives both a ready-to-propose task and a ready-to-code task
- **THEN** the proposal runner for the ready-to-propose task may execute concurrently with the Apply runner for the ready-to-code task

### Requirement: Archive orchestration uses ready-to-archive tasks
The system SHALL provide an Archive orchestration stage that loads managed tasks through `TaskManager` and processes tasks whose workflow state ID matches `LINEAR_STATE_READY_TO_ARCHIVE_ID`.

#### Scenario: Ready-to-archive task is selected
- **WHEN** `TaskManager` returns a task whose state ID equals `LINEAR_STATE_READY_TO_ARCHIVE_ID`
- **THEN** the orchestration stage treats that task as eligible for Archive execution

#### Scenario: Non-archive managed task is skipped by Archive route
- **WHEN** `TaskManager` returns a task from another managed state such as ready-to-propose or ready-to-code
- **THEN** the Archive route does not call the Archive runner for that task

#### Scenario: Proposal Apply and Archive tasks are processed in one pass
- **WHEN** one orchestration pass receives ready-to-propose, ready-to-code, and ready-to-archive tasks
- **THEN** the system routes each task to the executor matching its current state
- **AND** the system allows proposal and Apply processing to run concurrently
- **AND** the system processes Archive tasks sequentially without running Archive concurrently with proposal or Apply processing
