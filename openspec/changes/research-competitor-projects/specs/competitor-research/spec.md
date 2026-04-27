## ADDED Requirements

### Requirement: Competitor research captures source scope
The project SHALL provide a competitor research report that records the source GitHub organization page, the research date, and the exact repositories included in the analysis.

#### Scenario: Research source is traceable
- **WHEN** a reader opens the competitor research report
- **THEN** the report identifies `https://github.com/orgs/podlodka-ai-club/repositories?type=all` as the source page
- **AND** includes the date when the repository list was collected

#### Scenario: Repository list is explicit
- **WHEN** the research report summarizes the analyzed projects
- **THEN** it lists every repository included in the research with its GitHub URL
- **AND** it explicitly notes any repository skipped from deeper analysis with the reason

### Requirement: Competitor projects are analyzed with a consistent matrix
The project SHALL analyze each included competitor repository using a consistent set of comparison criteria relevant to the orchestration product.

#### Scenario: Repository analysis uses shared criteria
- **WHEN** a repository is included in the competitor research report
- **THEN** the report records its apparent purpose, primary stack, repository structure, user workflow, task or issue handling, agent or automation approach, git or PR workflow, observability, configuration approach, and test strategy when those signals are available

#### Scenario: Missing signals are represented explicitly
- **WHEN** a criterion cannot be determined from public repository materials
- **THEN** the report marks that criterion as not found or unclear instead of omitting it silently

### Requirement: Research extracts applicable ideas for the orchestrator
The project SHALL derive a shortlist of ideas that could improve the current orchestrator and separate those recommendations from raw repository observations.

#### Scenario: Idea shortlist is prioritized
- **WHEN** the research report presents applicable ideas
- **THEN** each idea includes the source repository or repositories, expected benefit, estimated implementation complexity, main risk, and recommended priority

#### Scenario: Ideas map to current project boundaries
- **WHEN** an idea is recommended for future work
- **THEN** the report maps it to the relevant current boundary such as `TaskManager`, `CoreOrch`, `AgentExecutor`, `GitManager`, `Logger`, CLI, configuration, tests, or documentation

#### Scenario: Follow-up changes remain separate
- **WHEN** a recommended idea requires runtime behavior changes
- **THEN** the report describes it as a candidate follow-up OpenSpec change rather than implementing it as part of the research change

### Requirement: Research report is reviewable as a project artifact
The project SHALL store the competitor research as a markdown artifact in the repository documentation so it can be reviewed through the normal proposal and PR workflow.

#### Scenario: Report is stored in documentation
- **WHEN** the research implementation is complete
- **THEN** the repository contains a markdown report under `docs/research/`

#### Scenario: Report includes enough detail for review
- **WHEN** a reviewer reads the report without re-running the research
- **THEN** they can see the source links, comparison matrix, key observations, recommendation shortlist, and limitations of the research
