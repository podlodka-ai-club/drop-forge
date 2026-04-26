## ADDED Requirements

### Requirement: Public cat-inspired project name
The project SHALL define `Purrch` as the public, human-facing project name for documentation and visual identity while keeping `orchv3` as the current technical CLI/module name.

#### Scenario: Developer opens project documentation
- **WHEN** a developer opens the root project documentation
- **THEN** the documentation presents `Purrch` as the project name
- **AND** it explains that the current executable/technical name remains `orchv3`

#### Scenario: Existing automation depends on orchv3
- **WHEN** existing scripts invoke the current CLI as `orchv3`
- **THEN** this branding change does not require those scripts to change

### Requirement: Repository logo asset
The repository SHALL include a cat-inspired logo asset for `Purrch` that can be rendered directly from tracked files without network access or an additional build step.

#### Scenario: README renders logo
- **WHEN** the root README references the project logo
- **THEN** the logo resolves to a tracked repository asset
- **AND** it can render without external image hosting

#### Scenario: Logo communicates project theme
- **WHEN** a reader sees the logo in documentation
- **THEN** the logo conveys a cat-inspired orchestration/coding-agent theme rather than a generic animal mark

### Requirement: Branding usage guidance
The repository SHALL document the intended usage boundary between `Purrch` and `orchv3`.

#### Scenario: Contributor updates documentation
- **WHEN** a contributor adds or edits project documentation
- **THEN** they can determine when to use `Purrch` as the public name and when to use `orchv3` as the technical CLI/module name

#### Scenario: Contributor considers technical rename
- **WHEN** a contributor wants to rename binaries, Go modules, package paths, environment variables, or repository names
- **THEN** the branding guidance identifies that work as a separate migration from this branding proposal
