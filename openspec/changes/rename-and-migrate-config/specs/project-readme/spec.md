## MODIFIED Requirements

### Requirement: Root README for project entrypoint
The repository SHALL contain a root-level `README.md` that explains what `Catmarine` is and what problem the current CLI solves. The README SHALL treat `Catmarine` as the current public product name and SHALL mention `orchv3` only as a legacy name or compatibility entrypoint when that context is needed.

#### Scenario: New developer opens README
- **WHEN** a developer opens `README.md`
- **THEN** `README.md` gives a concise description of the project purpose and the current proposal/apply/archive orchestration workflow under the `Catmarine` name

#### Scenario: README identifies the CLI entrypoint
- **WHEN** a developer looks for how to run the project
- **THEN** `README.md` shows the primary `Catmarine` CLI command or `go run ./cmd/catmarine`
- **AND** it identifies any retained `orchv3` command only as a compatibility path

#### Scenario: README avoids stale product naming
- **WHEN** a developer scans the top-level README title, introduction, setup, run instructions, and key directories
- **THEN** those sections use the new product name instead of presenting `orchv3` as the active name

### Requirement: README documents dependencies and configuration
The root `README.md` SHALL list the required development and runtime dependencies for the current workflow, SHALL describe the role of `.env` and `.env.example`, SHALL identify the external CLI prerequisites used by the application, and SHALL document the migration from legacy `PROPOSAL_*` keys to primary `CATMARINE_*` keys.

#### Scenario: Configure local environment
- **WHEN** a developer follows setup instructions
- **THEN** `README.md` points to `.env.example` as the template of supported environment variables and explains that actual values belong in `.env`

#### Scenario: Required tools are documented
- **WHEN** a developer reads prerequisites
- **THEN** `README.md` lists Go, `git`, `codex`, `gh`, and the requirement for authenticated GitHub access to the target repository

#### Scenario: Primary configuration keys are documented
- **WHEN** a developer reads the configuration section
- **THEN** `README.md` lists the primary `CATMARINE_*` keys used for repository, branch, command path, cleanup, and polling settings

#### Scenario: Legacy configuration aliases are documented
- **WHEN** a developer already has a `.env` using `PROPOSAL_*`
- **THEN** `README.md` explains that those keys are deprecated aliases
- **AND** it provides the mapping to the corresponding `CATMARINE_*` keys

#### Scenario: Configuration precedence is documented
- **WHEN** a developer sets both a `CATMARINE_*` key and its legacy `PROPOSAL_*` alias
- **THEN** `README.md` explains that the `CATMARINE_*` value takes precedence

### Requirement: README includes development navigation and verification commands
The root `README.md` SHALL include at least one section that helps developers navigate the repository or continue local development, such as key directories, links to detailed docs, or the standard verification commands used in the project. The navigation SHALL use the current `Catmarine` paths and mention legacy paths only when they remain intentionally supported.

#### Scenario: Developer needs standard checks
- **WHEN** a developer looks for how to verify local changes
- **THEN** `README.md` includes the standard project commands for formatting and tests

#### Scenario: Developer looks for source entrypoints
- **WHEN** a developer reads the key directories section
- **THEN** `README.md` lists `cmd/catmarine` as the primary CLI entrypoint
- **AND** it identifies `cmd/orchv3` only if that compatibility wrapper still exists
