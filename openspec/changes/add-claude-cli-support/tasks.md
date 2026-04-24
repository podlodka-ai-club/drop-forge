## 1. Configuration

- [ ] 1.1 Add proposal agent selection to `internal/config` with `PROPOSAL_AGENT_CLI`, default `codex`, and supported values `codex` and `claude`.
- [ ] 1.2 Add `PROPOSAL_CLAUDE_PATH` with default `claude` while keeping `PROPOSAL_CODEX_PATH` compatible with existing setups.
- [ ] 1.3 Update config validation so unknown agent values fail before side effects and only the selected agent executable path is required.
- [ ] 1.4 Update `.env.example` with the new agent selection and Claude path keys without values.
- [ ] 1.5 Add table-driven config tests for default Codex selection, Claude selection, unknown agent validation, and empty active agent path validation.

## 2. Agent Backend Abstraction

- [ ] 2.1 Introduce an internal proposal runner agent backend abstraction for executable path, argv, prompt, log module, and final response extraction.
- [ ] 2.2 Move current Codex command construction into the Codex backend without changing its default argv or prompt behavior.
- [ ] 2.3 Rename shared Codex-specific helper names only where they now represent selected-agent behavior, keeping compatibility where tests or public behavior depend on names.
- [ ] 2.4 Update runner flow to resolve the selected backend once after config validation and use it for prompt logging, command execution, output logging, and final response comment text.

## 3. Claude CLI Execution

- [ ] 3.1 Implement Claude argv builder for non-interactive print mode with JSON output and permission prompts disabled for the temp clone workflow.
- [ ] 3.2 Implement Claude prompt builder that instructs the agent to create OpenSpec proposal artifacts in the current workspace, follow project instructions, and stop before implementation.
- [ ] 3.3 Capture Claude stdout for final response extraction while continuing to forward stdout and stderr as JSON Lines log events.
- [ ] 3.4 Implement tolerant Claude final response extraction from JSON output with fallback to the last non-empty stdout content.
- [ ] 3.5 Ensure Claude command failures return errors that identify the Claude step and include logged process output.

## 4. Documentation

- [ ] 4.1 Update `docs/proposal-runner.md` with `PROPOSAL_AGENT_CLI`, Codex default behavior, Claude prerequisites, and the supported Claude command mode.
- [ ] 4.2 Update README only if it references the proposal runner prerequisites or environment variables.

## 5. Verification

- [ ] 5.1 Add proposal runner tests for Codex default behavior to prove existing command order and PR comment behavior remain unchanged.
- [ ] 5.2 Add proposal runner happy path test for Claude command name, argv, working directory, stdin prompt, git/PR continuation, and PR comment from Claude output.
- [ ] 5.3 Add proposal runner tests for empty Claude final response, unknown agent config, active agent path validation, and Claude command failure.
- [ ] 5.4 Run `go fmt ./...`.
- [ ] 5.5 Run `go test ./...`.
