## ADDED Requirements

### Requirement: Git provider is runtime configurable
The system SHALL read the review request provider from `.env` and environment variables using `PROPOSAL_GIT_PROVIDER`. The supported values SHALL be `github` and `gitlab`. When the variable is absent, the system SHALL use `github`.

#### Scenario: GitHub provider is the default
- **WHEN** `PROPOSAL_GIT_PROVIDER` is absent
- **THEN** configuration loading selects the GitHub provider
- **AND** existing GitHub workflows remain valid without adding new environment variables

#### Scenario: GitLab provider is configured
- **WHEN** `PROPOSAL_GIT_PROVIDER=gitlab`
- **THEN** configuration loading selects the GitLab provider
- **AND** provider-specific review request operations use GitLab semantics

#### Scenario: Unsupported provider is rejected
- **WHEN** `PROPOSAL_GIT_PROVIDER` contains a value other than `github` or `gitlab`
- **THEN** configuration validation returns an error before cloning or running external provider commands

### Requirement: Provider CLI paths are validated for selected provider
The system SHALL keep `PROPOSAL_GH_PATH` for GitHub operations and SHALL add `PROPOSAL_GLAB_PATH` for GitLab operations. Validation SHALL require the CLI path for the selected provider and SHALL NOT require the unselected provider CLI path.

#### Scenario: GitHub mode validates gh path
- **WHEN** the selected provider is `github`
- **THEN** configuration validation requires non-empty `PROPOSAL_GH_PATH`
- **AND** does not require `PROPOSAL_GLAB_PATH`

#### Scenario: GitLab mode validates glab path
- **WHEN** the selected provider is `gitlab`
- **THEN** configuration validation requires non-empty `PROPOSAL_GLAB_PATH`
- **AND** does not require `PROPOSAL_GH_PATH`

#### Scenario: Environment example lists provider keys
- **WHEN** a developer opens `.env.example`
- **THEN** it includes `PROPOSAL_GIT_PROVIDER` and `PROPOSAL_GLAB_PATH` without default values or secrets

### Requirement: Review request terminology is provider neutral
The system SHALL treat GitHub pull requests and GitLab merge requests as review requests at orchestration boundaries while preserving provider-specific command names and URLs in logs and errors.

#### Scenario: GitHub review request URL is accepted
- **WHEN** a runner returns a GitHub pull request URL
- **THEN** orchestration attaches that URL to the Linear task as the review request for the task

#### Scenario: GitLab review request URL is accepted
- **WHEN** a runner returns a GitLab merge request URL
- **THEN** orchestration attaches that URL to the Linear task as the review request for the task

