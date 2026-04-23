## ADDED Requirements

### Requirement: Codex final response PR comment
The system SHALL publish the last non-empty Codex response as a separate comment on the created pull request.

#### Scenario: Final Codex response is present
- **WHEN** the workflow creates a pull request and `codex exec` produced a non-empty last message
- **THEN** the system publishes that message as a pull request comment and logs the comment creation step

#### Scenario: Final Codex response is empty
- **WHEN** the workflow creates a pull request and the captured last Codex message is empty or whitespace-only
- **THEN** the system does not create an empty pull request comment and still returns the pull request URL

#### Scenario: Codex response comment fails
- **WHEN** the pull request is created but publishing the last Codex message as a comment fails
- **THEN** the system returns an error that identifies the comment step and logs the comment creation output

## REMOVED Requirements

### Requirement: Open questions PR comment
**Reason**: PR comment больше не должен зависеть от markdown-секций `Open Questions` внутри сгенерированных OpenSpec-артефактов.
**Migration**: Capture the final Codex message from `codex exec` and publish that message as the PR comment instead of scanning `openspec/changes/**/*.md`.
