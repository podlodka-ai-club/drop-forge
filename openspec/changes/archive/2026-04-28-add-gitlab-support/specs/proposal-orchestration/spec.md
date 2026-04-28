## ADDED Requirements

### Requirement: Orchestration preserves provider-neutral review request URLs
The system SHALL treat review request URLs returned by proposal execution and task attachments as opaque provider URLs, so GitHub pull request URLs and GitLab merge request URLs can flow through Linear without provider-specific orchestration logic.

#### Scenario: Proposal attaches GitHub URL
- **WHEN** proposal execution returns a GitHub pull request URL
- **THEN** the orchestration stage asks `TaskManager` to attach that URL to the task
- **AND** does not inspect or rewrite the URL

#### Scenario: Proposal attaches GitLab URL
- **WHEN** proposal execution returns a GitLab merge request URL
- **THEN** the orchestration stage asks `TaskManager` to attach that URL to the task
- **AND** does not inspect or rewrite the URL

#### Scenario: Apply receives provider URL
- **WHEN** a ready-to-code task has an attached GitHub pull request URL or GitLab merge request URL
- **THEN** the orchestration stage passes the URL to the Apply runner as the task branch source
- **AND** provider-specific branch resolution remains inside the runner repository dependency

#### Scenario: Archive receives provider URL
- **WHEN** a ready-to-archive task has an attached GitHub pull request URL or GitLab merge request URL
- **THEN** the orchestration stage passes the URL to the Archive runner as the task branch source
- **AND** provider-specific branch resolution remains inside the runner repository dependency

